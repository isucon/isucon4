package main

import (
	"os/exec"
	"strconv"
	"time"
)

type Slave struct {
	Master   *Master
	NowQueue *Queue
}

func NewSlave(master *Master) *Slave {
	return &Slave{
		Master: master,
	}
}

func (s *Slave) Waiting() {
	defer func() {
		if err := recover(); err != nil {
			defaultLogger.Info("%s", err)
		}
	}()

	for {
		if queue := s.Master.Pop(); queue != nil {
			s.Bench(queue)
		}
	}
}

func (s *Slave) Bench(queue *Queue) {
	s.NowQueue = queue
	now := time.Now()
	queue.StartedAt = &now

	logger := NewWSLogger(queue.ws)
	defaultLogger.Info("ベンチマーク開始: %s", queue.ApiKey)
	logger.Info("ベンチマークを開始します")

	cmd := exec.Command(
		"./benchmarker-2", "bb", "--hosts", queue.Option.Hosts, "--workload", strconv.Itoa(queue.Option.Workload), "--api-key", ``+queue.ApiKey+``,
	)

	cmd.Stdout = &WSLogger{"stdout", queue.ws}
	cmd.Stderr = &WSLogger{"stderr", queue.ws}

	err := cmd.Run()
	if err != nil {
		defaultLogger.Info("実行エラー: %s", err)
	}

	defaultLogger.Info("ベンチマーク完了: %s", queue.ApiKey)

	queue.ws.Close()
	s.NowQueue = nil
}
