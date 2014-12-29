package main

import (
	"fmt"
	"github.com/docker/docker/pkg/namesgenerator"
	"net/http"
)

type Advertiser struct {
	Id        int
	Slots     []*Slot
	Validated bool
}

func GetAdvertiser() *Advertiser {
	return &Advertiser{
		Slots:     []*Slot{},
		Validated: false,
	}
}

func (ad *Advertiser) Apply(req *http.Request) {
	req.Header.Set("X-Advertiser-Id", fmt.Sprintf("%d", ad.Id))
}

func (ad *Advertiser) NewSlot() *Slot {
	s := NewSlot(fmt.Sprintf("%d-%s", ad.Id, namesgenerator.GetRandomName(0)))
	s.Advertiser = ad
	ad.Slots = append(ad.Slots, s)
	return s
}

func (adr *Advertiser) AllAds() []*Ad {
	ads := []*Ad{}

	for _, s := range adr.Slots {
		ads = append(ads, s.Ads...)
	}

	return ads
}
