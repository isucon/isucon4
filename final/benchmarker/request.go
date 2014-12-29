package main

import (
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (w *Worker) NewRequest(method, uri string, body io.Reader) (*http.Request, error) {
	parsedURL, err := url.ParseRequestURI(uri)

	if err != nil {
		return nil, err
	}

	if parsedURL.Scheme == "" {
		parsedURL.Scheme = "http"
	}

	if parsedURL.Host == "" {
		parsedURL.Host = w.Host()
	}

	req, err := http.NewRequest(strings.ToUpper(method), parsedURL.String(), body)

	if err != nil {
		return nil, err
	}

	return req, err
}

func (w *Worker) SendRequest(
	req *http.Request,
	to time.Duration,
) (resp *http.Response, err error) {
	reqCh := make(chan bool)

	cache := w.Recipe.URLCaches[req.URL.String()]
	if cache != nil {
		cache.Apply(req)
	}

	w.nowRequest = req

	req.Header.Set("Connection", "Keep-Alive")

	go func() {
		resp, err = w.Client.Do(req)
		reqCh <- true
	}()

	timeoutTimer := time.After(to)
	waitingResponse := true

	for waitingResponse {
		select {
		case <-reqCh:
			waitingResponse = false
		case <-timeoutTimer:
			w.Transport.CancelRequest(req)
			if w.running {
				err = ErrRequestTimeout
			} else {
				err = ErrRequestCanceled
			}
			waitingResponse = false
		}
	}

	w.nowRequest = nil

	if err == nil && resp != nil {
		if resp.StatusCode == http.StatusNotModified && cache != nil {
			cache.Restore(resp)
		} else {
			if resp.Request.Method == "GET" && resp.StatusCode >= http.StatusOK && resp.StatusCode <= 299 {
				cache := NewCache(resp)
				if cache != nil && w != nil && w.Recipe != nil && w.Recipe.URLCaches != nil {
					w.Recipe.URLCaches[req.URL.String()] = cache
				}
			}
		}
	}

	return resp, err
}
