package main

import (
	"crypto/sha256"
	"fmt"
	"github.com/martini-contrib/sessions"
	"io"
	"os"
)

func getEnv(key string, def string) string {
	v := os.Getenv(key)
	if len(v) == 0 {
		return def
	}

	return v
}

func getFlash(session sessions.Session, key string) string {
	value := session.Get(key)

	if value == nil {
		return ""
	} else {
		session.Delete(key)
		return value.(string)
	}
}

func calcPassHash(password, hash string) string {
	h := sha256.New()
	io.WriteString(h, password)
	io.WriteString(h, ":")
	io.WriteString(h, hash)

	return fmt.Sprintf("%x", h.Sum(nil))
}
