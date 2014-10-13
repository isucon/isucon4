package worker

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

var (
	ErrRequestTimeout  = errors.New("Connection timeout")
	ErrRequestCanceled = errors.New("Request cancelled because benchmark finished (1min)")
)

func (w *Worker) SimpleGet(uri string) (*http.Request, *http.Response, error) {
	req, err := w.NewRequest("GET", uri, nil)

	if err != nil {
		return req, nil, err
	}

	res, err := w.SendRequest(req, true)

	return req, res, err
}

func (w *Worker) NewRequest(method, uri string, body io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		parsedURL.Host = w.Host
	}

	req, err := http.NewRequest(method, parsedURL.String(), body)

	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Forwarded-For", w.IPList.Get().String())

	return req, err
}

func (w *Worker) SendRequest(
	req *http.Request, simple bool,
) (resp *http.Response, err error) {
	reqCh := make(chan bool)

	if !simple {
		w.nowRequest = req
	}

	req.Header.Set("User-Agent", UserAgent)

	if w.Debug {
		w.logger.Printf("type:request\tid:%d\tmethod:%s\turi:%s", w.ID, req.Method, req.URL)
	}

	startedAt := time.Now()

	go func() {
		resp, err = w.Client.Do(req)
		reqCh <- true
	}()

	timeoutTimer := time.After(w.TimeoutDuration)
	waitingResponse := true

	for waitingResponse {
		select {
		case <-reqCh:
			waitingResponse = false
		case <-timeoutTimer:
			w.Transport.CancelRequest(req)
			if w.Running {
				err = ErrRequestTimeout
			} else {
				err = ErrRequestCanceled
			}
			waitingResponse = false
		}
	}

	if !simple {
		w.nowRequest = nil
	}

	finishedAt := time.Now()
	elapsedMsec := int64(finishedAt.Sub(startedAt) / time.Millisecond)

	if err == nil {
		if w.Debug {
			w.logger.Printf(
				"type:response\tid:%d\tmethod:%s\turi:%s\ttimeout:false\tstatus:%d\telapsed:%v",
				w.ID,
				resp.Request.Method,
				resp.Request.URL,
				resp.StatusCode,
				elapsedMsec,
			)
		}
	}

	return resp, err
}
