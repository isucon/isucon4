package main

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
)

type BenchmarkRecipe struct {
	Hosts             []string `json:"hosts"`
	Workload          int      `json:"workload"`
	advertisers       []*Advertiser
	users             []*User
	advertiserWorkers []*Worker
	userWorkers       []*Worker
	adrIdx            int
	astIdx            int
	URLCaches         map[string]*URLCache
	SendScore         bool
	logger            *Logger
	ApiKey            string
	NoForce           bool
}

func NewRecipe() *BenchmarkRecipe {
	br := &BenchmarkRecipe{
		Hosts:     []string{},
		Workload:  DEFAULT_WORKLOAD,
		URLCaches: map[string]*URLCache{},
	}

	br.generateUsers()

	return br
}

func (br *BenchmarkRecipe) NewSlot() (*Slot, *Worker) {
	var adr *Advertiser
	if len(br.advertisers) < MaxAdvertisersCount {
		adr = br.AddNewAdvertiser()
	} else {
		adr = br.advertisers[br.adrIdx%MaxAdvertisersCount]
		br.adrIdx++
	}

	slot := adr.NewSlot()
	slot.Assets = GetAssets(br.astIdx)
	br.astIdx++

	w := NewWorker()
	w.Recipe = br
	w.Hosts = br.Hosts
	w.Role = AdvertiserWorker
	w.Advertiser = adr
	w.Slot = slot
	w.DummyServer = httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set(ValidationHeaderKey, ValidationHeaderVal)

		if _, ok := w.Slot.pathAndAd[r.URL.Path[1:]]; ok {
			rw.WriteHeader(http.StatusNoContent)
		} else {
			rw.WriteHeader(http.StatusNotFound)
		}
	}))
	w.logger = br.logger
	br.advertiserWorkers = append(br.advertiserWorkers, w)

	for i := 0; i < (br.Workload * 2); i++ {
		w := NewWorker()
		w.Recipe = br
		w.Hosts = br.Hosts
		w.Role = UserWorker
		w.Advertiser = adr
		w.Slot = slot
		w.logger = br.logger
		br.userWorkers = append(br.userWorkers, w)
	}

	return slot, w
}

func (br *BenchmarkRecipe) AddNewAdvertiser() *Advertiser {
	advertiser := GetAdvertiser()
	advertiser.Id = len(br.advertisers) + 1
	br.advertisers = append(br.advertisers, advertiser)
	return advertiser
}

func (br *BenchmarkRecipe) generateUsers() {
	for i := 0; i < GenerateUsersCount; i++ {
		u := GetRandomUser()
		br.users = append(br.users, u)
	}
}

func (br *BenchmarkRecipe) GetRandomUser() *User {
	return br.users[rand.Intn(len(br.users))]
}

func (br *BenchmarkRecipe) WakeupWorkers(wg *sync.WaitGroup) {
	for _, w := range br.advertiserWorkers {
		if !w.running {
			wg.Add(1)
			go func(w *Worker) {
				w.Wait()
				wg.Done()
			}(w)
			w.Run()
		}
	}

	for _, w := range br.userWorkers {
		if !w.running {
			wg.Add(1)
			go func(w *Worker) {
				w.Wait()
				wg.Done()
			}(w)
			w.Run()
		}
	}
}

func (br *BenchmarkRecipe) Abort() {
	var wg sync.WaitGroup
	for _, w := range append(br.advertiserWorkers, br.userWorkers...) {
		wg.Add(1)
		go func(w *Worker) {
			w.Abort()
			wg.Done()
		}(w)
	}
	wg.Wait()
}

func (br *BenchmarkRecipe) ValidateReports() {
	var wg sync.WaitGroup
	for _, w := range br.advertiserWorkers {
		wg.Add(1)
		go func(w *Worker) {
			w.running = true
			w.ValidateReport()
			w.running = false
			wg.Done()
		}(w)
	}
	wg.Wait()
}

func (br *BenchmarkRecipe) ErrorReport() ErrorReport {
	errs := []*BenckmarkError{}

	for _, w := range br.advertiserWorkers {
		errs = append(errs, w.Errors...)
	}

	for _, w := range br.userWorkers {
		errs = append(errs, w.Errors...)
	}

	return ErrorReport(errs)
}
