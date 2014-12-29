package main

import (
	"net/http"
)

type Requester interface {
	Apply(*http.Request)
}
