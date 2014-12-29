package main

import (
	"sync"
)

type Slot struct {
	*sync.Mutex

	Advertiser *Advertiser
	Id         string
	Ads        []*Ad
	Assets     []*Asset
	astIdx     int
	idAndAd    map[string]*Ad
	pathAndAd  map[string]*Ad
}

func NewSlot(id string) *Slot {
	return &Slot{
		Mutex:     new(sync.Mutex),
		Id:        id,
		idAndAd:   map[string]*Ad{},
		pathAndAd: map[string]*Ad{},
	}
}

func (s *Slot) NewAd(url string) *Ad {
	ad := NewAd()
	ad.Advertiser = s.Advertiser
	ad.Slot = s
	ad.Asset = s.Assets[s.astIdx%len(s.Assets)]
	s.astIdx++
	ad.Destination = url
	return ad
}
