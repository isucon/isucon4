package main

import (
	"bytes"
	"github.com/marcw/cachecontrol"
	"io/ioutil"
	"net/http"
	"time"
)

type ClosableBuffer struct {
	*bytes.Reader
}

func (b *ClosableBuffer) Close() error {
	b.Reader = nil
	return nil
}

type URLCache struct {
	LastModified   string
	Etag           string
	ExpiresAt      time.Time
	CacheControl   *cachecontrol.CacheControl
	CachedResponse *http.Response
	CachedBody     []byte
	MD5            string
}

func NewCache(res *http.Response) *URLCache {
	directive := res.Header.Get("Cache-Control")
	cc := cachecontrol.Parse(directive)
	noCache, _ := cc.NoCache()

	if len(directive) == 0 || noCache || cc.NoStore() {
		return nil
	}

	now := time.Now()
	lm := res.Header.Get("Last-Modified")
	etag := res.Header.Get("ETag")

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil
	}
	res.Body.Close()
	md5 := GetMD5(body)

	if len(body) >= 1024*1024*4 {
		body = []byte{}
	}

	res.Body = &ClosableBuffer{bytes.NewReader(body)}

	res.Header.Set(CachedHeader, CachedHeaderVal)
	res.Header.Set(CachedMD5Header, md5)

	return &URLCache{
		LastModified:   lm,
		Etag:           etag,
		ExpiresAt:      now.Add(cc.MaxAge()),
		CacheControl:   &cc,
		CachedResponse: res,
		CachedBody:     body,
		MD5:            md5,
	}
}

func (c *URLCache) Available() bool {
	return time.Now().Before(c.ExpiresAt)
}

func (c *URLCache) Apply(req *http.Request) {
	if c.Available() {
		if c.LastModified != "" {
			req.Header.Add("If-Modified-Since", c.LastModified)
		}

		if c.Etag != "" {
			req.Header.Add("If-None-Match", c.Etag)
		}
	}
}

func (c *URLCache) Restore(res *http.Response) {
	res.Status = c.CachedResponse.Status
	res.StatusCode = c.CachedResponse.StatusCode
	res.Header = c.CachedResponse.Header
	res.ContentLength = c.CachedResponse.ContentLength
	res.TransferEncoding = c.CachedResponse.TransferEncoding
	res.Body = &ClosableBuffer{bytes.NewReader(c.CachedBody)}
	res.Header.Set(CachedHeader, CachedHeaderVal)
	res.Header.Set(CachedMD5Header, c.MD5)
}
