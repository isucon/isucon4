package main

import (
	"code.google.com/p/go.net/websocket"
	"github.com/codegangsta/cli"
	"io"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func getMyIP() string {
	ifaces, _ := net.Interfaces()
	for _, nic := range ifaces {
		addrs, _ := nic.Addrs()
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), "10.") {
				ads := strings.Split(addr.String(), "/")

				if len(ads) > 0 {
					return ads[0]
				}
			}
		}
	}

	return ""
}

var remoteFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "hosts, H",
		Usage:  "ベンチ実行先ホスト。カンマ区切りで複数設定可能。",
		EnvVar: "HOSTS",
		Value:  getMyIP(),
	},
	cli.IntFlag{
		Name:   "workload, w",
		Usage:  "並列ワーカー数。1 から 8 の範囲で指定。",
		EnvVar: "WORKLOAD",
		Value:  int(DEFAULT_WORKLOAD),
	},
}

func RemoteAction(c *cli.Context) {
	if ApiKey == "None" {
		defaultLogger.Info("API-KEY が設定されていません。環境変数 ISUCON_API_KEY を確認するか、運営へご相談ください。")
		os.Exit(1)
	}

	team := GetTeamByApiKey(ApiKey)
	if team == nil {
		defaultLogger.Info("チーム情報の取得に失敗しました。API-KEY がおかしい可能性があります。確認の上、運営へご相談ください。 API-KEY: %s", ApiKey)
		os.Exit(1)
	}

	ws, err := websocket.Dial("ws://"+MasterHost+"/ws", "", "http://"+MasterIP+"/")
	if err != nil {
		defaultLogger.Info("%s", err)
		os.Exit(1)
	}
	defer ws.Close()

	sigC := make(chan os.Signal)
	signal.Notify(sigC, syscall.SIGINT)
	go func() {
		for {
			select {
			case <-sigC:
				websocket.JSON.Send(ws, &RemoteCommand{
					Name: "cancel",
					Options: map[string]interface{}{
						"api-key": ApiKey,
					}})
			}
		}
	}()

	alive := true
	closed := make(chan bool)

	go func() {
		for alive {
			var cmd *RemoteCommand
			err := websocket.JSON.Receive(ws, &cmd)
			if err != nil {
				if err == io.EOF {
					break
				}

				defaultLogger.Info("%s", err)
			} else {
				cmd.Execute(ws)
			}
		}
		closed <- true
	}()

	go func() {
		for alive {
			time.Sleep(1 * time.Second)
			websocket.JSON.Send(ws, &RemoteCommand{Name: "ping"})
		}
	}()

	websocket.JSON.Send(ws, &RemoteCommand{
		Name: "bench",
		Options: map[string]interface{}{
			"team-id":  team.Id,
			"api-key":  ApiKey,
			"hosts":    c.String("hosts"),
			"workload": c.Int("workload"),
		},
	})

	<-closed
}
