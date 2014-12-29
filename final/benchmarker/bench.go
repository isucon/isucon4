package main

import (
	"encoding/json"
	"github.com/codegangsta/cli"
	"log"
	"net/http"
	_ "net/http/pprof"
	"strings"
	"sync"
	"time"
)

var benchFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "hosts, H",
		Usage:  "ベンチ実行先ホスト。カンマ区切りで複数設定可能。",
		EnvVar: "HOSTS",
		Value:  "127.0.0.1",
	},
	cli.IntFlag{
		Name:   "workload, w",
		Usage:  "並列ワーカー数。1 から 8 の範囲で指定。",
		EnvVar: "WORKLOAD",
		Value:  int(DEFAULT_WORKLOAD),
	},
}

func BenchAction(c *cli.Context) {
	if Version == "debug" {
		go func() {
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	recipe := NewRecipe()

	hosts := strings.Split(c.String("hosts"), ",")
	for _, host := range hosts {
		host = strings.TrimSpace(host)

		if len(host) > 0 {
			recipe.Hosts = append(recipe.Hosts, host)
		}
	}

	if c.Int("workload") > 0 && c.Int("workload") <= 8 {
		recipe.Workload = c.Int("workload")
	}

	recipe.SendScore = MasterAPIKey != "None"

	Bench(recipe, nil)
}

func Bench(recipe *BenchmarkRecipe, logger *Logger) {
	if logger == nil {
		logger = defaultLogger
	}

	recipe.logger = logger

	defer func() {
		if err := recover(); err != nil {
			log.Fatal("%s", err)
		}
	}()

	logger.Info("アセットデータを事前ロード中です...")
	PreloadAssets(AssetsDir)

	wg := new(sync.WaitGroup)
	running := true

	logger.Info("初期化エンドポイントへ POST リクエストを送信しています...")

	http.Post("http://"+recipe.Hosts[0]+"/initialize", "text/plain", nil)

	logger.Info("初期化完了")

	bmt := time.Now()
	logger.Info("ベンチマーク開始")

	go func() {
		for {
			select {
			case <-time.After(workerRunningDuration()):
				running = false
				recipe.Abort()
				return
			}
		}
	}()

	wakeuped := false
	go func() {
		for running {
			_, w := recipe.NewSlot()
			recipe.WakeupWorkers(wg)
			wakeuped = true

			for !w.allAds {
				time.Sleep(1 * time.Second)
			}
		}
	}()

	for !wakeuped {
		time.Sleep(1 * time.Second)
	}

	wg.Wait()

	logger.Info("ベンチマーク完了(%s)", time.Now().Sub(bmt))

	time.Sleep(20 * time.Second)

	logger.Info("レポートの検証開始")
	recipe.ValidateReports()
	logger.Info("レポートの検証完了")

	logger.Info("結果を JSON 形式で標準出力へ書き出します")

	scTotal, scSucc, scFail, data := recipe.Score()
	jData, _ := json.MarshalIndent(data, "", "  ")

	logger.Info("得点: %.2f (加点: %.2f / 減点: %.2f)", scTotal, scSucc, scFail)

	if recipe.SendScore {
		logger.Info("スコアの送信中...")
		err := SendScore(recipe.ApiKey, scTotal, scSucc, scFail)
		if err != nil {
			logger.Info("スコアの送信が正常に行われませんでした: %s", err)
		}
	}

	logger.Write(jData)
}
