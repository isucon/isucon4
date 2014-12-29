package main

import (
	"github.com/docker/docker/pkg/namesgenerator"
	"strconv"
	"sync"
	"sync/atomic"
)

type Ad struct {
	*sync.Mutex

	Advertiser   *Advertiser
	Slot         *Slot
	Id           string
	Title        string
	Destination  string
	Path         string
	Asset        *Asset
	Impression   int64
	ClickedUsers []*User
}

func NewAd() *Ad {
	return &Ad{
		Mutex:        new(sync.Mutex),
		Title:        namesgenerator.GetRandomName(0),
		Path:         namesgenerator.GetRandomName(0),
		Impression:   0,
		ClickedUsers: []*User{},
	}
}

func (ad *Ad) IncrImp() {
	atomic.AddInt64(&ad.Impression, 1)
}

func (ad *Ad) Click(u *User) {
	ad.Lock()
	defer ad.Unlock()

	ad.ClickedUsers = append(ad.ClickedUsers, u)
}

func (ad *Ad) BreakDown() (agents, gender, generations map[string]float64) {
	ad.Lock()
	defer ad.Unlock()

	agents = map[string]float64{}
	gender = map[string]float64{}
	generations = map[string]float64{}

	for _, u := range ad.ClickedUsers {
		agents[u.UserAgent]++

		if u.DNT {
			gender["unknown"]++
			generations["unknown"]++
		} else {
			sex := "female"
			if u.Gender != 0 {
				sex = "male"
			}
			gender[sex]++
			generations[strconv.Itoa(int(u.Age/10))]++
		}
	}

	return
}
