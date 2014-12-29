package main

import (
	"code.google.com/p/go.net/websocket"
	"time"
)

type Queue struct {
	ws        *websocket.Conn
	TeamId    int          `json:"team_id"`
	ApiKey    string       `json:"-"`
	QueuedAt  time.Time    `json:"queued_at"`
	StartedAt *time.Time   `json:"started_at"`
	Option    *QueueOption `json:"options"`
}

type QueueOption struct {
	Hosts    string `json:"hosts"`
	Workload int    `json:"workload"`
}
