package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ErrLevel int

const (
	ErrFatal ErrLevel = iota
	ErrError
	ErrNotice
)

func (lv ErrLevel) String() string {
	prefix := "NOTICE"
	switch lv {
	case ErrFatal:
		prefix = "FATAL"
	case ErrError:
		prefix = "ERROR"
	}

	return prefix
}

type Stringifier interface {
	String() string
}

type BenckmarkError struct {
	Level   ErrLevel
	URL     string
	Request *http.Request
	Error   string
	On      time.Time
}

func NewError(lv ErrLevel, url string, err interface{}, req *http.Request) *BenckmarkError {

	if lv != ErrFatal && err == ErrRequestTimeout {
		lv = ErrNotice
	}

	if err == ErrRequestCanceled {
		lv = ErrNotice
	}

	errString := fmt.Sprintf("%v", err)

	if e, ok := err.(Stringifier); ok {
		errString = e.String()
	}

	if e, ok := err.(error); ok {
		errString = e.Error()
	}

	return &BenckmarkError{
		Level:   lv,
		URL:     url,
		Request: req,
		Error:   errString,
		On:      time.Now(),
	}
}

func (e *BenckmarkError) String() string {
	return fmt.Sprintf("[%s / %s] %s", e.On.Format(TimeFormat), e.Level.String(), e.Error)
}

func StatusCodeMissMatch(expect, got int) error {
	return errors.New(
		fmt.Sprintf(
			"HTTP Status code miss match: expected '%d', got '%d'",
			expect, got,
		),
	)
}

func MD5MissMatch(expect, got string) error {
	return errors.New(
		fmt.Sprintf(
			"MD5 hash miss match: expected '%s', got '%s'",
			expect, got,
		),
	)
}

type ErrorReport []*BenckmarkError

func (e ErrorReport) IsSuccess() bool {
	suc := true
	for _, err := range e {
		if err.Level == ErrFatal {
			suc = false
			break
		}
	}

	return suc
}

func (e ErrorReport) ToJSON() map[string][]string {
	j := map[string][]string{}

	for _, err := range e {
		errs, hit := j[err.URL]
		if !hit {
			errs = []string{}
		}
		j[err.URL] = append(errs, err.String())
	}

	return j
}

func (e ErrorReport) Write(w io.Writer) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(e.ToJSON())
}
