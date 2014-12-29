package main

import (
	"bytes"
	"code.google.com/p/go.net/websocket"
	"container/list"
	"encoding/json"
	"github.com/codegangsta/cli"
	"io"
	"net/http"
	_ "net/http/pprof"
	"sync"
	"time"
)

var masterFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "port, p",
		EnvVar: "PORT",
		Value:  "9091",
	},
	cli.IntFlag{
		Name:   "slave-count",
		EnvVar: "SLAVE_COUNT",
		Value:  1,
	},
}

type Master struct {
	*sync.Mutex

	list *list.List

	server *http.Server
	slaves []*Slave
}

var ServerMaster *Master

func MasterAction(c *cli.Context) {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()

	defaultLogger.Info("アセットの事前ロード開始")
	PreloadAssets(AssetsDir)
	defaultLogger.Info("アセット事前ロード完了")

	master := &Master{
		Mutex:  new(sync.Mutex),
		list:   list.New(),
		slaves: []*Slave{},
	}
	master.NewServer(":" + c.String("port"))

	for i := 0; i < c.Int("slave-count"); i++ {
		slave := NewSlave(master)
		master.slaves = append(master.slaves, slave)
		go func() {
			slave.Waiting()
		}()
	}

	ServerMaster = master
	defaultLogger.Info("サーバー起動: %s", master.server.Addr)

	go func() {
		for {
			time.Sleep(1 * time.Second)
			master.UpdateQueues()
		}
	}()

	master.server.ListenAndServe()
}

func (m *Master) NewServer(bind string) {

	mux := http.NewServeMux()

	mux.Handle("/ws", websocket.Handler(m.wsHandler))

	m.server = &http.Server{Addr: bind, Handler: mux}
	m.server.SetKeepAlivesEnabled(true)
}

func (m *Master) wsHandler(ws *websocket.Conn) {
	defer func() {
		if err := recover(); err != nil {
			defaultLogger.Info("%s", err)
		}
	}()

	defer ws.Close()

	for {
		var cmd *RemoteCommand
		err := websocket.JSON.Receive(ws, &cmd)
		if err != nil {
			if err != io.EOF {
				defaultLogger.Info("%s", err)
			}
			return

		} else {
			cmd.Execute(ws)
		}
	}
}

func (m *Master) Push(ws *websocket.Conn, teamId int, apiKey, hosts string, workload int) bool {
	if m.AlreadQueued(apiKey) {
		return false
	}

	m.Lock()
	defer m.Unlock()
	m.list.PushBack(&Queue{
		ws:       ws,
		TeamId:   teamId,
		ApiKey:   apiKey,
		QueuedAt: time.Now(),
		Option: &QueueOption{
			Hosts:    hosts,
			Workload: workload,
		},
	})

	return true
}

func (m *Master) Pop() *Queue {
	m.Lock()
	defer m.Unlock()

	elm := m.list.Front()
	if elm == nil {
		return nil
	}

	queue, ok := elm.Value.(*Queue)
	m.list.Remove(elm)

	if !ok {
		return nil
	}

	return queue
}

func (m *Master) AlreadQueued(apiKey string) bool {
	m.Lock()
	defer m.Unlock()

	already := false

	for e := m.list.Front(); e != nil; e = e.Next() {
		queue, ok := e.Value.(*Queue)
		if ok && queue.ApiKey == apiKey {
			already = true
		}
	}

	return already
}

func (m *Master) Cancel(apiKey string) {

	m.Lock()
	defer m.Unlock()

	for e := m.list.Front(); e != nil; e = e.Next() {
		queue, ok := e.Value.(*Queue)
		if ok && queue.ApiKey == apiKey {
			m.list.Remove(e)
			WSInfo(queue.ws, "キャンセルされました")
			queue.ws.Close()
			return
		}
	}
}

func (m *Master) QueueInfo() map[string][]*Queue {
	m.Lock()
	defer m.Unlock()

	running := []*Queue{}
	pending := []*Queue{}

	for e := m.list.Front(); e != nil; e = e.Next() {
		queue, ok := e.Value.(*Queue)
		if ok {
			pending = append(pending, queue)
		}
	}

	for _, s := range m.slaves {
		if s.NowQueue != nil {
			running = append(running, s.NowQueue)
		}
	}

	return map[string][]*Queue{
		"running": running,
		"pending": pending,
	}
}

func (m *Master) UpdateQueues() {
	blob, err := json.Marshal(m.QueueInfo())
	if err != nil {
		defaultLogger.Info("UpdateQueue ERR %s", err)
		return
	}

	jsonBody := bytes.NewReader(blob)

	req, err := http.NewRequest("POST", "https://isucon4-portal.herokuapp.com/queue/update", jsonBody)
	if err != nil {
		defaultLogger.Info("UpdateQueue ERR %s", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "isucon "+MasterAPIKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		defaultLogger.Info("UpdateQueue ERR %s", err)
		return
	}
	defer res.Body.Close()

	if res.StatusCode != 204 {
		defaultLogger.Info("UpdateQueue ERR StatusCode is %d", res.StatusCode)
		return
	}
}
