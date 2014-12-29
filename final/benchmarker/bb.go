package main

import (
	"github.com/codegangsta/cli"
	"os"
	"strings"
)

var bbFlags = []cli.Flag{
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
	cli.StringFlag{
		Name:  "api-key",
		Value: "None",
	},
}

func BBAction(c *cli.Context) {
	recipe := NewRecipe()

	hosts := strings.Split(c.String("hosts"), ",")
	for _, host := range hosts {
		host = strings.TrimSpace(host)

		if len(host) > 0 {
			recipe.Hosts = append(recipe.Hosts, host)
		}
	}

	if len(recipe.Hosts) < 1 {
		defaultLogger.Info("実行先ホストが存在しないためベンチが実行できません")
		os.Exit(1)
	}

	if c.Int("workload") > 0 && c.Int("workload") <= 8 {
		recipe.Workload = c.Int("workload")
	}

	if c.Int("workload") >= 8 {
		recipe.Workload = 8
	}

	ApiKey = c.String("api-key")

	if ApiKey == "None" {
		defaultLogger.Info("API-KEY が設定されていません。環境変数 ISUCON_API_KEY を確認するか、運営へご相談ください。")
		os.Exit(1)
	}

	team := GetTeamByApiKey(ApiKey)
	if team == nil {
		defaultLogger.Info("チーム情報の取得に失敗しました。API-KEY がおかしい可能性があります。確認の上、運営へご相談ください。 API-KEY: %s", ApiKey)
		os.Exit(1)
	}

	recipe.ApiKey = ApiKey
	recipe.SendScore = true
	recipe.NoForce = true

	Bench(recipe, nil)
}
