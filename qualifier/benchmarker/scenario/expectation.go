package scenario

import (
	"net/http"
	"time"
)

type Expectation struct {
	StatusCode    int
	Location      string
	Headers       map[string]string
	Selectors     []string
	Assets        map[string]string
	HTML          map[string]string
	LastLoginedAt time.Time
	Checksum      string
}

func (e Expectation) Check(res *http.Response) error {
	return nil
}
