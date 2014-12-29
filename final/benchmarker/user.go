package main

import (
	"fmt"
	"math/rand"
	"net/http"
)

type User struct {
	Gender    int
	Age       int
	UserAgent string
	DNT       bool
}

func GetRandomUser() *User {
	return &User{
		Gender:    rand.Int() % 2,
		Age:       UserUnderAge + rand.Intn(UserUpperAge),
		UserAgent: GetUserAgent(),
		DNT:       (rand.Int() % UserDNTPercentage) == 0,
	}
}

func (u *User) Identifier() string {
	return fmt.Sprintf("%d/%d", u.Gender, u.Age)
}

func (u *User) Apply(req *http.Request) {
	if !u.DNT {
		cookie := &http.Cookie{
			Name:   "isuad",
			Value:  u.Identifier(),
			Path:   "/",
			Domain: req.URL.Host,
		}
		req.AddCookie(cookie)
	}
	req.Header.Set("User-Agent", u.UserAgent)
}
