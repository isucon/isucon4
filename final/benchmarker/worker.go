package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"sync"
	"time"
)

type WorkerRole uint

func workerRunningDuration() time.Duration {
	d, err := time.ParseDuration(WORKER_RUNNING_DURATION)

	if err != nil {
		return 1 * time.Minute
	}

	return d
}

func workerTimeoutDuration() time.Duration {
	d, err := time.ParseDuration(WORKER_TIMEOUT_DURATION)

	if err != nil {
		return 10 * time.Second
	}

	return d
}

func workerAbortingDuration() time.Duration {
	d, err := time.ParseDuration(WORKER_ABORTING_DURATION)

	if err != nil {
		return 10 * time.Second
	}

	return d
}

type Worker struct {
	*sync.Mutex

	Recipe          *BenchmarkRecipe
	Hosts           []string
	Client          *http.Client
	Transport       *http.Transport
	TimeoutDuration time.Duration
	Role            WorkerRole
	Advertiser      *Advertiser
	User            *User
	Slot            *Slot
	DummyServer     *httptest.Server
	Errors          []*BenckmarkError

	nowRequest *http.Request
	running    bool
	stopped    chan bool
	abortChan  chan bool
	hostsIdx   int
	allAds     bool
	logger     *Logger
}

func NewWorker() *Worker {
	w := &Worker{
		Mutex:           new(sync.Mutex),
		TimeoutDuration: workerTimeoutDuration(),
		Errors:          []*BenckmarkError{},

		running:   false,
		hostsIdx:  0,
		stopped:   make(chan bool),
		abortChan: make(chan bool),
		allAds:    false,
	}

	jar, _ := cookiejar.New(&cookiejar.Options{})
	w.Transport = &http.Transport{
		ResponseHeaderTimeout: 3 * time.Minute,
		MaxIdleConnsPerHost:   16,
		DisableKeepAlives:     false,
		DisableCompression:    false,
	}
	w.Client = &http.Client{
		Transport: w.Transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}

			req.Header.Set("Connection", "Keep-Alive")
			req.Header.Set("User-Agent", via[0].Header.Get("User-Agent"))

			return nil
		},
	}

	return w
}

func (w *Worker) Run() {
	w.running = true

	go func() {
		defer func() {
			w.stopped <- true
		}()

		jq := make(chan bool, 1)
		jq <- true

		for {
			select {
			case <-w.abortChan:
				return
			case <-jq:
				go func() {
					w.Work()

					if w.running {
						jq <- true
					}
				}()
			}
		}
	}()
}

func (w *Worker) Abort() {
	w.running = false
	w.abortChan <- true

	time.Sleep(workerAbortingDuration())
	if w.nowRequest != nil {
		w.Transport.CancelRequest(w.nowRequest)
	}
}

func (w *Worker) Wait() {
	<-w.stopped

	go func() {
		time.Sleep(30 * time.Second)
		if w.DummyServer != nil {
			w.DummyServer.CloseClientConnections()
			w.DummyServer.Close()
			w.DummyServer = nil
		}
	}()
}

func (w *Worker) Host() string {
	if len(w.Hosts) < 1 {
		return ""
	}

	defer func() { w.hostsIdx++ }()
	return w.Hosts[w.hostsIdx%len(w.Hosts)]
}

func (w *Worker) Work() {
	switch w.Role {
	case AdvertiserWorker:
		w.WorkAdvertiser()
	case UserWorker:
		w.WorkUser()
	}
}

func (w *Worker) Do(req *http.Request, to time.Duration) (*http.Response, error) {
	var requester Requester

	switch w.Role {
	case AdvertiserWorker:
		requester = w.Advertiser
	case UserWorker:
		requester = w.User
	}

	if requester != nil {
		requester.Apply(req)
	}

	resp, err := w.SendRequest(req, to)

	return resp, err
}

func (w *Worker) JSONDo(req *http.Request, to time.Duration) (*http.Response, map[string]interface{}, error) {
	var value map[string]interface{}

	resp, err := w.Do(req, to)

	if err != nil {
		return resp, value, err
	}
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(&value)

	if err != nil {
		return resp, value, err
	}

	return resp, value, err
}

func (w *Worker) AddError(err *BenckmarkError) {
	w.Lock()
	defer w.Unlock()

	if w.running {
		w.Errors = append(w.Errors, err)
		if w.logger != nil {
			w.logger.Info("エラー: %s : %s", err.URL, err.String())
		}
	}
}
