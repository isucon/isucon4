package worker

import (
	"errors"
	"fmt"
	"github.com/isucon/isucon4/qualifier/benchmarker/ip"
	"github.com/isucon/isucon4/qualifier/benchmarker/user"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"sync/atomic"
	"time"
)

const (
	UserAgent              = "ISUCON4 Benchmarker"
	MaxInvalidResponse     = 10
	DefaultTimeoutDuration = 10 * time.Second
)

var latestWorkerID = 0

type Worker struct {
	ID              int
	Client          *http.Client
	Transport       *http.Transport
	TimeoutDuration time.Duration

	Host   string
	IPList *ip.IPList
	Users  []*user.User

	Successes int32
	Fails     int32
	Score     int64
	Errors    []error

	Running  bool
	Debug    bool
	FastFail bool

	stoppedChan chan struct{}
	logger      *log.Logger

	nowRequest *http.Request
}

func New() *Worker {
	latestWorkerID++

	w := &Worker{
		ID:              latestWorkerID,
		TimeoutDuration: DefaultTimeoutDuration,

		IPList: ip.NextIPList(),
		Users:  user.GetDummyUsers(100),

		Debug:    false,
		Running:  false,
		FastFail: true,

		logger: log.New(os.Stdout, "", 0),
	}
	w.logger.SetFlags(log.Ltime)

	jar, _ := cookiejar.New(&cookiejar.Options{})
	w.Transport = &http.Transport{}
	w.Client = &http.Client{
		Transport: w.Transport,
		Jar:       jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("stopped after 10 redirects")
			}

			req.Header.Set("User-Agent", via[0].Header.Get("User-Agent"))
			req.Header.Set("X-Forwarded-For", via[0].Header.Get("X-Forwarded-For"))

			return nil
		},
	}

	return w
}

func (w *Worker) Success(point int64) {
	atomic.AddInt32(&w.Successes, 1)
	atomic.AddInt64(&w.Score, point)
}

func (w *Worker) Fail(req *http.Request, err error) error {
	atomic.AddInt32(&w.Fails, 1)
	if req != nil {
		err = fmt.Errorf("%s\tmethod:%s\turi:%s", err, req.Method, req.URL.Path)
	}

	if w.FastFail {
		w.logger.Printf("type:fail\treason:%s", err)
	}

	w.Errors = append(w.Errors, err)
	return err
}

func (w *Worker) Reset() {
	jar, _ := cookiejar.New(&cookiejar.Options{})
	w.Client.Jar = jar
}
