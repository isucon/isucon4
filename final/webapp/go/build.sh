#!/bin/bash

go get github.com/go-martini/martini
go get gopkg.in/redis.v3
go get github.com/martini-contrib/render
go get github.com/kr/pretty
go build -o golang-webapp .
