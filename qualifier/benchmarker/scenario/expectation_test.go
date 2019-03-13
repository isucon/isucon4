package scenario

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"
)

var mockFormHTML = `
<div class="container">
  <form class="form-horizontal" role="form" action="/login" method="POST">
    <div class="form-group">
      <label for="input-username" class="col-sm-3 control-label">お客様ご契約ID</label>
      <div class="col-sm-9">
        <input id="input-username" type="text" class="form-control" placeholder="半角英数字" name="login">
      </div>
    </div>
    <div class="form-group">
      <label for="input-password" class="col-sm-3 control-label">パスワード</label>
      <div class="col-sm-9">
        <input type="password" class="form-control" id="input-password" name="password" placeholder="半角英数字・記号（２文字以上）">
      </div>
    </div>
    <div class="form-group">
      <div class="col-sm-offset-3 col-sm-9">
        <button type="submit" class="btn btn-primary btn-lg btn-block">ログイン</button>
      </div>
    </div>
  </form>
</div>`

var mockLoginHTML = `
<dl class="dl-horizontal">
  <dt>前回ログイン</dt>
  <dd id="last-logined-at">2019-03-13 18:13:45</dd>
  <dt>最終ログインIPアドレス</dt>
  <dd id="last-logined-ip">127.0.0.1</dd>
</dl>`

func TestCheck(t *testing.T) {
	lastLogin, _ := time.ParseInLocation("2006-01-02 15:04:05", "2019-03-13 18:13:45", time.Local)
	cases := []struct {
		sut   Expectation
		input *http.Response
		want  error
	}{
		{
			sut:   Expectation{StatusCode: 123},
			input: NewResponseBuilder().SetStatusCode(123).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{StatusCode: 100},
			input: NewResponseBuilder().SetStatusCode(101).Build(),
			want:  fmt.Errorf("Response code should be 100, got 101"),
		},
		{
			sut:   Expectation{Location: ""}, // skip the check if empty
			input: NewResponseBuilder().SetLocation("/test").Build(),
			want:  nil,
		},
		{
			sut:   Expectation{Location: "/test"},
			input: NewResponseBuilder().SetLocation("/test").Build(),
			want:  nil,
		},
		{
			sut:   Expectation{Location: "/test"},
			input: NewResponseBuilder().SetLocation("/wrong_path").Build(),
			want:  fmt.Errorf("Expected location is miss match /test, got: /wrong_path"),
		},
		{
			sut:   Expectation{Headers: map[string]string{"key": "value"}},
			input: NewResponseBuilder().SetHeaders(map[string]string{"key": "value"}).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{Headers: map[string]string{"key": "value"}},
			input: NewResponseBuilder().Build(),
			want:  fmt.Errorf("Expected header is miss match: value, got "),
		},
		{
			sut:   Expectation{Selectors: []string{"input[name='login']"}},
			input: NewResponseBuilder().SetBody(mockFormHTML).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{Selectors: []string{"*[type='submit']"}},
			input: NewResponseBuilder().SetBody(mockFormHTML).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{Selectors: []string{"input[name='username']"}},
			input: NewResponseBuilder().SetBody(mockFormHTML).Build(),
			want:  fmt.Errorf("Expected selector is not found: input[name='username']"),
		},
		{
			sut:   Expectation{HTML: map[string]string{"#last-logined-ip": "127.0.0.1"}},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{HTML: map[string]string{"#unknown-id": "127.0.0.1"}},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  fmt.Errorf("Expected selector is not found: #unknown-id"),
		},
		{
			sut:   Expectation{HTML: map[string]string{"#last-logined-ip": "invalid"}},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  fmt.Errorf("Expected html text is match: invalid, got 127.0.0.1"),
		},
		{
			sut:   Expectation{LastLoginedAt: lastLogin},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  nil,
		},
		{
			sut:   Expectation{LastLoginedAt: lastLogin},
			input: NewResponseBuilder().Build(),
			want:  fmt.Errorf("Expected selector is not found: #last-logined-at"),
		},
		{
			sut:   Expectation{LastLoginedAt: lastLogin.Add(2 * time.Second)},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  fmt.Errorf("Expected last logined time is match: 2019-03-13 18:13:47, got 2019-03-13 18:13:45"),
		},
		{
			sut:   Expectation{LastLoginedAt: lastLogin.Add(-2 * time.Second)},
			input: NewResponseBuilder().SetBody(mockLoginHTML).Build(),
			want:  fmt.Errorf("Expected last logined time is match: 2019-03-13 18:13:43, got 2019-03-13 18:13:45"),
		},
	}
	for i, c := range cases {
		res := c.sut.Check(c.input)
		switch {
		case res == nil && c.want != nil:
			t.Errorf("#%d: Check(%v): got nil, want %q", i, c.input, c.want.Error())
		case res != nil && c.want == nil:
			t.Errorf("#%d: Check(%v): got %q, want nil", i, c.input, res.Error())
		case res != nil && c.want != nil:
			if got, want := res.Error(), c.want.Error(); got != want {
				t.Errorf("#%d: Check(%v): got %q, want %q", i, c.input, got, want)
			}
		}
	}
}

type ResponseBuilder struct {
	StatusCode int
	Location   string
	Headers    map[string]string
	Body       string
}

func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{}
}

func (rb *ResponseBuilder) SetStatusCode(code int) *ResponseBuilder {
	rb.StatusCode = code
	return rb
}

func (rb *ResponseBuilder) SetLocation(loc string) *ResponseBuilder {
	rb.Location = loc
	return rb
}

func (rb *ResponseBuilder) SetHeaders(hs map[string]string) *ResponseBuilder {
	rb.Headers = hs
	return rb
}

func (rb *ResponseBuilder) SetBody(s string) *ResponseBuilder {
	rb.Body = s
	return rb
}

func (rb *ResponseBuilder) Build() *http.Response {
	header := http.Header{}
	for key, val := range rb.Headers {
		header.Add(key, val)
	}

	var buf bytes.Buffer
	buf.Write([]byte(rb.Body))

	return &http.Response{
		Header:     header,
		StatusCode: rb.StatusCode,
		Body:       ioutil.NopCloser(&buf),
		Request:    &http.Request{URL: &url.URL{Path: rb.Location}},
	}
}
