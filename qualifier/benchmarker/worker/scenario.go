package worker

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"github.com/isucon/isucon4/qualifier/benchmarker/ip"
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/html"
	"github.com/moovweb/gokogiri/xml"
	"io"
	"io/ioutil"
	"net/url"
	"sync"
	"time"
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

	ExpectedStatusCode    int
	ExpectedLocation      string
	ExpectedHeaders       map[string]string
	ExpectedSelectors     []string
	ExpectedAssets        map[string]string
	ExpectedHTML          map[string]string
	ExpectedLastLoginedAt time.Time
	ExpectedChecksum      string
}

func NewScenario(method, path string) *Scenario {
	return &Scenario{
		Method: method,
		Path:   path,

		ExpectedStatusCode: 200,
		ExpectedHeaders:    map[string]string{},
		ExpectedSelectors:  []string{},
		ExpectedAssets:     map[string]string{},
		ExpectedChecksum:   "",
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

	if res.StatusCode != s.ExpectedStatusCode {
		return w.Fail(res.Request, fmt.Errorf("Response code should be %d, got %d", s.ExpectedStatusCode, res.StatusCode))
	}

	if s.ExpectedLocation != "" {
		if s.ExpectedLocation != res.Request.URL.Path {
			return w.Fail(
				res.Request,
				fmt.Errorf(
					"Expected location is miss match %s, got: %s",
					s.ExpectedLocation, res.Request.URL.Path,
				))
		}
	}

	body, _ := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	doc, err := gokogiri.ParseHtml(body)
	defer doc.Free()

	if err != nil {
		return w.Fail(res.Request, fmt.Errorf("Invalid html document"))
	}

	for header, value := range s.ExpectedHeaders {
		respHeader := res.Header.Get(header)
		if respHeader != value {
			return w.Fail(res.Request, fmt.Errorf("Expected header is miss match: %s, got %s", respHeader, value))
		}
	}

	for _, selector := range s.ExpectedSelectors {
		nodes, err := doc.Search(selector)
		if err != nil {
			return w.Fail(res.Request, fmt.Errorf("node search error"))
		}

		if len(nodes) == 0 {
			return w.Fail(res.Request, fmt.Errorf("Expected selector is not found: %s", selector))
		}
	}

	for selector, innerHTML := range s.ExpectedHTML {
		nodes, err := doc.Search(selector)
		if err != nil {
			return w.Fail(res.Request, fmt.Errorf("node search error"))
		}

		if len(nodes) == 0 {
			return w.Fail(res.Request, fmt.Errorf("Expected selector is not found: %s", selector))
		}

		if nodes[0].InnerHtml() != innerHTML {
			return w.Fail(res.Request, fmt.Errorf(
				"Expected html text is match: %s, got %s",
				innerHTML, nodes[0].InnerHtml(),
			))
		}
	}

	if !s.ExpectedLastLoginedAt.IsZero() {
		selector := "//*[@id='last-logined-at']"
		nodes, err := doc.Search(selector)
		if err != nil {
			return w.Fail(res.Request, fmt.Errorf("node search error"))
		}

		if len(nodes) == 0 {
			return w.Fail(res.Request, fmt.Errorf("Expected selector is not found: %s", selector))
		}

		parsedTime, err := time.ParseInLocation(
			"2006-01-02 15:04:05", nodes[0].InnerHtml(), time.Local,
		)

		if err != nil {
			return w.Fail(res.Request, fmt.Errorf("Parse time error: %s", err))
		}

		if s.ExpectedLastLoginedAt.Add(1*time.Second).Before(parsedTime) &&
			s.ExpectedLastLoginedAt.Add(-1*time.Second).After(parsedTime) {
			return w.Fail(res.Request, fmt.Errorf(
				"Expected last logined time is match: %s, got %s",
				s.ExpectedLastLoginedAt.Format("2006-01-02 15:04:05"),
				nodes[0].InnerHtml(),
			))
		}
	}

	s.CheckAssets(w, doc)

	w.Success(ScenarioScore)
	return nil
}

func (s *Scenario) CheckAssets(w *Worker, doc *html.HtmlDocument) {
	var wg sync.WaitGroup

	base, err := url.Parse(s.Path)

	if err != nil {
		return
	}

	// <link>
	links, err := doc.Search("//link")
	if err == nil {
		for _, link := range links {
			if link.Attr("href") != "" {
				wg.Add(1)
				go func(link xml.Node) {
					s.GetAsset(w, base, link, "href")
					wg.Done()
				}(link)
			}
		}
	}

	// <script>
	scripts, err := doc.Search("//script")
	if err == nil {
		for _, script := range scripts {
			if script.Attr("src") != "" {
				wg.Add(1)
				go func(script xml.Node) {
					s.GetAsset(w, base, script, "src")
					wg.Done()
				}(script)
			}
		}
	}

	// img
	imgs, err := doc.Search("//img")
	if err == nil {
		for _, img := range imgs {
			if img.Attr("src") != "" {
				wg.Add(1)
				go func(img xml.Node) {
					s.GetAsset(w, base, img, "src")
					wg.Done()
				}(img)
			}
		}
	}

	wg.Wait()
}

func (s *Scenario) GetAsset(w *Worker, base *url.URL, node xml.Node, attr string) error {
	path, err := url.Parse(node.Attr(attr))

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

	if expectedMD5, ok := s.ExpectedAssets[requestURI.RequestURI()]; ok {
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
