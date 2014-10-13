package worker

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	. "testing"
	"time"
)

func TestWorkerExecuteRequest(t *T) {
	worker := New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client")
	}))
	defer ts.Close()

	resp, err := worker.SimpleGet(ts.URL)

	if err != nil {
		t.Fatal(err)
	}

	responseText, err := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if err != nil {
		t.Fatal(err)
	}

	if bytes.Compare(responseText, []byte("Hello, client")) == 0 {
		t.Fatal(string(responseText))
	}
}

func TestWorkerRequestWithTimeout(t *T) {
	worker := New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(11 * time.Second)
		fmt.Fprintln(w, "Client timeout")
	}))
	defer ts.Close()

	resp, err := worker.SimpleGet(ts.URL)

	if err != ErrRequestTimeout {
		t.Fatal(resp, err)
	}
}
