package worker

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/KentaKudo/isucon4/qualifier/benchmarker/ip"
	"github.com/KentaKudo/isucon4/qualifier/benchmarker/scenario"
	"github.com/PuerkitoBio/goquery"
)

const (
	ScenarioScore   = 100
	StaticFileScore = 2
)

type Scenario struct {
	Method string
	Path   string
	IP     *ip.IP

	PostData map[string]string
	Headers  map[string]string

	Expectation scenario.Expectation
}

func NewScenario(method, path string) *Scenario {
	return &Scenario{
		Method: method,
		Path:   path,

		Expectation: scenario.Expectation{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{},
			Selectors:  []string{},
			Assets:     map[string]string{},
		},
	}
}

func (s *Scenario) Play(w *Worker) error {
	formData := url.Values{}
	for key, val := range s.PostData {
		formData.Set(key, val)
	}

	buf := bytes.NewBufferString(formData.Encode())
	req, err := w.NewRequest(s.Method, s.Path, buf)

	if err != nil {
		return w.Fail(req, err)
	}

	for key, val := range s.Headers {
		req.Header.Add(key, val)
	}
	if s.IP != nil {
		req.Header.Set("X-Forwarded-For", s.IP.String())
	}
	if req.Method != "GET" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	res, err := w.SendRequest(req, false)
	if err != nil {
		return w.Fail(req, err)
	}

	if err := s.Expectation.Check(res); err != nil {
		return w.Fail(res.Request, err)
	}

	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return w.Fail(res.Request, fmt.Errorf("Invalid html document"))
	}

	s.CheckAssets(w, doc)

	w.Success(ScenarioScore)
	return nil
}

func (s *Scenario) CheckAssets(w *Worker, doc *goquery.Document) {
	var wg sync.WaitGroup

	base, err := url.Parse(s.Path)

	if err != nil {
		return
	}

	for tag, attr := range map[string]string{
		"link": "href", "script": "src", "img": "src",
	} {
		doc.Find(tag).Each(func(i int, sel *goquery.Selection) {
			if uri, ok := sel.Attr(attr); ok && uri != "" {
				wg.Add(1)
				go func() {
					s.GetAsset(w, base, uri)
					wg.Done()
				}()
			}
		})
	}

	wg.Wait()
}

func (s *Scenario) GetAsset(w *Worker, base *url.URL, relpath string) error {
	path, err := url.Parse(relpath)

	if err != nil {
		return w.Fail(nil, err)
	}

	requestURI := base.ResolveReference(path)

	req, res, err := w.SimpleGet(requestURI.String())

	if err != nil {
		return w.Fail(req, err)
	}

	if res.StatusCode != 200 {
		return w.Fail(res.Request, fmt.Errorf("Response code should be %d, got %d", 200, res.StatusCode))
	}

	md5sum := calcMD5(res.Body)
	defer res.Body.Close()

	if expectedMD5, ok := s.Expectation.Assets[requestURI.RequestURI()]; ok {
		if md5sum == expectedMD5 {
			w.Success(StaticFileScore)
		} else {
			return w.Fail(res.Request, fmt.Errorf("Expected MD5 checksum is miss match %s, got %s", expectedMD5, md5sum))
		}
	}

	return nil
}

func calcMD5(body io.Reader) string {
	h := md5.New()
	_, _ = io.Copy(h, body)

	return fmt.Sprintf("%x", h.Sum(nil))
}
