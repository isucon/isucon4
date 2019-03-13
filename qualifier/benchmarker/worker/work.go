package worker

import (
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/KentaKudo/isucon4/qualifier/benchmarker/ip"
	"github.com/KentaKudo/isucon4/qualifier/benchmarker/user"
)

func (w *Worker) Work() {
	w.Running = true
	w.stoppedChan = make(chan struct{})
	time.AfterFunc(1*time.Minute, func() {
		w.Timeout()
	})

	go func() {
		checker := 0
		for w.Running {
			if checker%20 == 0 {
				if w.IPList.IsAlmostBlacklisted() {
					w.IPList = ip.NextIPList()
				}

				if w.IsUsersAlmostBlackListed() {
					w.Users = user.GetDummyUsers(100)
				}
			}

			ip := w.IPList.Next()
			user := w.Users[rand.Intn(len(w.Users))]
			for user.InUse() {
				user = w.Users[rand.Intn(len(w.Users))]
			}

			user.Start()
			w.Login(ip, user)

			if !w.Running {
				user.FlagIncomplete()
				ip.FlagIncomplete()
			}

			user.Finish()

			checker++
		}

		w.stoppedChan <- struct{}{}
	}()
}

func (w *Worker) Timeout() {
	w.Running = false
	w.TimeoutDuration = 50 * time.Millisecond

	<-time.After(10 * time.Second)
	if w.nowRequest != nil {
		w.Transport.CancelRequest(w.nowRequest)
	}
}

func (w *Worker) Stop() {
	<-w.stoppedChan
}

func (w *Worker) IsUsersAlmostBlackListed() bool {
	blacklisted := 0

	for _, user := range w.Users {
		if user.IsBlacklisted() {
			blacklisted++
		}
	}

	num := len(w.Users) / 5

	return num < blacklisted
}

func (w *Worker) SendScore(apiKey string, score float64, successes, fails int32, metadata map[string]string) error {
	sendResult := NewScenario("POST", "/results")
	sendResult.Headers = map[string]string{
		"X-API-KEY": apiKey,
	}
	sendResult.PostData = map[string]string{
		"score":     strconv.Itoa(int(score)),
		"successes": strconv.Itoa(int(successes)),
		"fails":     strconv.Itoa(int(fails)),
	}
	for key, val := range metadata {
		sendResult.PostData["metadata["+key+"]"] = val
	}
	sendResult.Expectation.StatusCode = http.StatusCreated
	return sendResult.Play(w)
}
