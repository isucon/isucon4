package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"os"
	"time"
)

type Logger struct {
	Stdout io.Writer
	Stderr io.Writer
}

var defaultLogger = &Logger{os.Stdout, os.Stderr}

func (l *Logger) Write(b []byte) (int, error) {
	return l.Stdout.Write(b)
}

func (l *Logger) Info(msg string, args ...interface{}) {
	io.WriteString(
		l.Stderr,
		fmt.Sprintf(
			"[%s] %s\n",
			time.Now().Format(TimeFormat),
			fmt.Sprintf(msg, args...),
		),
	)
}

type WSLogger struct {
	To string
	ws *websocket.Conn
}

func (w *WSLogger) Write(b []byte) (int, error) {
	err := websocket.JSON.Send(w.ws, &RemoteCommand{
		Name: w.To,
		Options: map[string]interface{}{
			"body": string(b),
		},
	})

	return len(b), err
}

func NewWSLogger(ws *websocket.Conn) *Logger {
	return &Logger{
		Stdout: &WSLogger{"stdout", ws},
		Stderr: &WSLogger{"stderr", ws},
	}
}
