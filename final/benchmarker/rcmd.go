package main

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"io"
	"os"
	"time"
)

func Exists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

type RemoteCommand struct {
	Name    string                 `json:"name"`
	Options map[string]interface{} `json:"options"`
}

func (c *RemoteCommand) Execute(ws *websocket.Conn) {
	switch c.Name {
	case "ping":
		websocket.JSON.Send(ws, &RemoteCommand{Name: "pong"})
	case "cancel":
		ServerMaster.Cancel(c.Options["api-key"].(string))
	case "bench":
		if Exists("/home/isucon/stop-queue") {
			WSInfo(ws, "現在新規キューイング停止中のため追加されません")
			ws.Close()
			return
		}

		queued := ServerMaster.Push(
			ws,
			int(c.Options["team-id"].(float64)),
			c.Options["api-key"].(string),
			c.Options["hosts"].(string),
			int(c.Options["workload"].(float64)),
		)

		if !queued {
			WSInfo(ws, "すでに処理待ちキューに追加済みのため、追加されません")
			ws.Close()
		} else {
			WSInfo(ws, "実行待ちキューに追加されました")
		}

	case "stdout":
		body := c.Options["body"]

		sbody, ok := body.(string)
		if ok {
			io.WriteString(os.Stdout, sbody)
		}
	case "stderr":
		body := c.Options["body"]

		sbody, ok := body.(string)
		if ok {
			io.WriteString(os.Stderr, sbody)
		}
	}
}

func WSInfo(ws *websocket.Conn, msg string, args ...interface{}) {
	websocket.JSON.Send(ws, &RemoteCommand{
		Name: "stderr",
		Options: map[string]interface{}{
			"body": fmt.Sprintf(
				"[%s] %s\n",
				time.Now().Format(TimeFormat),
				fmt.Sprintf(msg, args...),
			),
		},
	})
}
