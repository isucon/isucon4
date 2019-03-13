package scenario

import (
	"fmt"
	"net/http"
	"testing"
)

func TestCheck(t *testing.T) {
	cases := []struct {
		sut   Expectation
		input *http.Response
		want  error
	}{
		{
			sut:   Expectation{StatusCode: 123},
			input: &http.Response{StatusCode: 123},
			want:  nil,
		},
		{
			sut:   Expectation{StatusCode: 100},
			input: &http.Response{StatusCode: 101},
			want:  fmt.Errorf("Response code should be 100, got 101"),
		},
	}
	for _, c := range cases {
		got := c.sut.Check(c.input)
		if got != c.want {
			t.Errorf("Check(%v): got %v, want %v", c.input, got, c.want)
		}
	}
}
