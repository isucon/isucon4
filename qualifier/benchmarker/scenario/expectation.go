package scenario

import (
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Expectation struct {
	StatusCode    int
	Location      string
	Headers       map[string]string
	Selectors     []string
	Assets        map[string]string
	HTML          map[string]string
	LastLoginedAt time.Time
}

func (e Expectation) Check(res *http.Response) error {
	if got, want := res.StatusCode, e.StatusCode; got != want {
		return fmt.Errorf("Response code should be %d, got %d", want, got)
	}

	if got, want := res.Request.URL.Path, e.Location; want != "" && got != want {
		return fmt.Errorf("Expected location is miss match %s, got: %s", want, got)
	}

	for h, v := range e.Headers {
		if got, want := res.Header.Get(h), v; got != want {
			return fmt.Errorf("Expected header is miss match: %s, got %s", want, got)
		}
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return fmt.Errorf("Invalid html document")
	}

	for _, selector := range e.Selectors {
		if doc.Find(selector).Length() == 0 {
			return fmt.Errorf("Expected selector is not found: %s", selector)
		}
	}

	for selector, innerHTML := range e.HTML {
		var err error
		len := doc.Find(selector).Each(func(i int, s *goquery.Selection) {
			if got, want := s.Text(), innerHTML; got != want {
				err = fmt.Errorf("Expected html text is match: %s, got %s", want, got)
			}
		}).Length()

		if len == 0 {
			return fmt.Errorf("Expected selector is not found: %s", selector)
		}

		if err != nil {
			return err
		}
	}

	if !e.LastLoginedAt.IsZero() {
		lastLoginStr := doc.Find("#last-logined-at").Text()
		if lastLoginStr == "" {
			return fmt.Errorf("Expected selector is not found: #last-logined-at")
		}

		format := "2006-01-02 15:04:05"
		lastLogin, err := time.ParseInLocation(format, lastLoginStr, time.Local)
		if err != nil {
			return fmt.Errorf("Parse time error: %s", err)
		}

		if got, want := lastLogin, e.LastLoginedAt; got.Before(want.Add(-1*time.Second)) || got.After(want.Add(1*time.Second)) {
			return fmt.Errorf("Expected last logined time is match: %s, got %s", want.Format(format), got.Format(format))
		}
	}

	return nil
}
