package main

import (
	"encoding/json"
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/isucon/isucon4/qualifier/benchmarker/ip"
	"github.com/isucon/isucon4/qualifier/benchmarker/user"
	"github.com/isucon/isucon4/qualifier/benchmarker/worker"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var (
	GIT_COMMIT       string
	DebugMode        string
	SkipMetadataMode string
)

const (
	PortalDomain = "isucon4-portal.herokuapp.com"
)

var (
	app          *cli.App
	logger       *log.Logger
	Debug        = false
	SkipMetadata = false
	defaultInit  string
)

func init() {
	Debug = DebugMode == "true"
	SkipMetadata = SkipMetadataMode == "true"

	logger = log.New(os.Stdout, "", 0)
	logger.SetFlags(log.Ltime)

	if Debug {
		logger.Print("type:info\tmessage:!!! DEBUG MODE !!! DEBUGE MODE !!!")

		defaultInit = "./init.sh"
	} else {
		defaultInit = "/home/isucon/init.sh"
	}

	app = cli.NewApp()
	app.Name = "benchmarker"
	app.Usage = "ISUCON4 benchmarker for qualifier"
	app.Version = "v2 " + GIT_COMMIT

	app.Commands = []cli.Command{
		{
			Name:      "bench",
			ShortName: "b",
			Usage:     "Run benchmark",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:   "api-key",
					Value:  "",
					Usage:  "Benchmark API Key",
					EnvVar: "ISUCON4_BENCH_API_KEY",
				},
				cli.IntFlag{
					Name:   "workload",
					Value:  1,
					Usage:  "Benchmark workload",
					EnvVar: "ISUCON4_BENCH_WORKLOAD",
				},
				cli.StringFlag{
					Name:   "init",
					Value:  defaultInit,
					Usage:  "Bench init script path",
					EnvVar: "ISUCON4_BENCH_INIT",
				},
				cli.StringFlag{
					Name:   "host",
					Value:  "localhost",
					Usage:  "Bench Endpoint host",
					EnvVar: "ISUCON4_BENCH_HOST",
				},
			},
			Action: benchmark,
		},
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	runtime.GOMAXPROCS(runtime.NumCPU())

	app.Run(os.Args)
}

func benchmark(c *cli.Context) {
	checkInstanceMetadata()

	logger.Print("type:info\tmessage:launch benchmarker")

	if c.String("api-key") == "" {
		logger.Printf("type:warning\tmessage:Result not sent to server because API key is not set")
	}

	initEnvironment(c)

	workload := c.Int("workload")
	if workload < 1 {
		workload = 1
	}

	workers := worker.Workers(make([]*worker.Worker, workload))

	for i := 0; i < workload; i++ {
		workers[i] = worker.New()
		workers[i].Host = c.String("host")
	}

	logger.Printf("type:info\tmessage:run benchmark workload: %d", workload)

	workers.Work()

	workers.Stop()

	logger.Printf("type:info\tmessage:finish benchmark workload: %d", workload)

	totalSuccesses := int32(0)
	totalFails := int32(0)
	totalScore := float64(0)

	<-time.After(5 * time.Second)

	for _, worker := range workers {
		totalSuccesses += worker.Successes
		totalFails += worker.Fails
		totalScore += math.Ceil(float64(worker.Score) / 100.0)

		//for _, err := range worker.Errors {
		//logger.Printf("type:fail\treason:%s", err)
		//}
	}

	err := checkReport(c)
	if err != nil {
		logger.Printf("%s\tmessage:Report checking is failed. Do not send score.", err)
		logger.Printf("type:score\tsuccess:%d\tfail:%d\tscore:%d", totalSuccesses, totalFails, int64(totalScore))
		os.Exit(1)
	}

	sendScore(c, totalScore, totalSuccesses, totalFails)

	logger.Printf("type:score\tsuccess:%d\tfail:%d\tscore:%d", totalSuccesses, totalFails, int64(totalScore))
}

func initEnvironment(c *cli.Context) {
	logger.Print("type:info\tmessage:init environment")

	var initErr error
	initCh := make(chan bool)

	go func() {
		userInit := exec.Command(c.String("init"))
		if err := userInit.Run(); err != nil {
			initErr = fmt.Errorf("reason:init script failed\tlog:%s", err)
		}
		initCh <- true
	}()

	waiting := true
	timer := time.After(1 * time.Minute)

	for waiting {
		select {
		case <-initCh:
			waiting = false
		case <-timer:
			waiting = false
			initErr = fmt.Errorf("reason:init script timed out")
		}
	}

	if initErr != nil {
		logger.Printf("type:fail\t%s\tmessage:Do not run benchmark", initErr)
		os.Exit(1)
	}
}

func checkReport(c *cli.Context) error {
	logger.Print("type:info\tmessage:check banned ips and locked users report")

	w := worker.New()
	w.Host = c.String("host")
	w.TimeoutDuration = 1 * time.Minute

	_, res, err := w.SimpleGet("/report")

	if err != nil {
		return fmt.Errorf("type:fail\treason:%s", err)
	}

	var report map[string][]string

	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		return fmt.Errorf("type:fail\treason:%s", err)
	}

	err = json.Unmarshal([]byte(body), &report)

	if err != nil {
		return fmt.Errorf("type:fail\treason:%s", err)
	}

	blacklistedIPs := []*ip.IP{}
	for _, ipList := range ip.GeneratedIPList {
		for _, ip := range ipList.All() {
			if ip.IsBlacklisted() {
				blacklistedIPs = append(blacklistedIPs, ip)
			}
		}
	}
	logger.Printf("type:report\tcount:banned ips\tvalue:%d", len(blacklistedIPs))

	blacklistedUsers := []*user.User{}
	for _, u := range user.DummyUsers {
		if u.IsBlacklisted() {
			blacklistedUsers = append(blacklistedUsers, u)
		}
	}
	logger.Printf("type:report\tcount:locked users\tvalue:%d", len(blacklistedUsers))

	matchBlacklistedIPs := len(blacklistedIPs) == len(report["banned_ips"])
	if matchBlacklistedIPs {
		for _, bip := range blacklistedIPs {
			m := false
			for _, rip := range report["banned_ips"] {
				if rip == bip.String() {
					m = true
				}
			}

			if Debug && bip.IsIncomplete() {
				var mStr string
				if m {
					mStr = "true"
				} else {
					mStr = "false"
				}
				logger.Printf("type:debug\tincomplete_ip:%s\treported:%s", bip.String(), mStr)
			}

			if bip.IsIncomplete() {
				m = true
			}

			matchBlacklistedIPs = matchBlacklistedIPs && m
		}
	}

	matchBlacklistedUsers := len(blacklistedUsers) == len(report["locked_users"])
	if matchBlacklistedUsers {
		for _, bu := range blacklistedUsers {
			m := false
			for _, ru := range report["locked_users"] {
				if ru == bu.Name {
					m = true
				}
			}

			if Debug && bu.IsIncomplete() {
				var mStr string
				if m {
					mStr = "true"
				} else {
					mStr = "false"
				}
				logger.Printf("type:debug\tincomplete_user:%s\treported:%s", bu.Name, mStr)
			}

			if bu.IsIncomplete() {
				m = true
			}

			matchBlacklistedUsers = matchBlacklistedUsers && m
		}
	}

	if !matchBlacklistedIPs {
		return fmt.Errorf("type:fail\treason:Missmatch banned IPs")
	}

	if !matchBlacklistedUsers {
		return fmt.Errorf("type:fail\treason:Missmatch banned Users")
	}

	return nil
}

func sendScore(c *cli.Context, score float64, successes, fails int32) {
	apiKey := c.String("api-key")

	if apiKey == "" {
		logger.Printf("type:info\tmessage:Result not sent to server because API key is not set")
		return
	}

	w := worker.New()
	w.Host = PortalDomain
	if Debug {
		w.Host = "localhost:3000"
	}
	w.TimeoutDuration = 1 * time.Minute

	metadata := map[string]string{
		"instance_type": instanceType,
		"instance_id":   instanceId,
		"ami_id":        amiId,
		"cpu_info":      cpuInfo,
	}
	err := w.SendScore(apiKey, score, successes, fails, metadata)
	if err != nil {
		logger.Printf("type:fail\tmessage: Score sending failed\treason:%s", err)
	}
}
