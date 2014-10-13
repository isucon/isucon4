package ip

import (
	"math/rand"
	"sync/atomic"
)

var cNum = 0

var GeneratedIPList []*IPList

type IPList struct {
	ips []*IP
	idx uint32
}

func NextIPList() *IPList {
	ipList := NewIPList(127, 1, byte(cNum))
	cNum++
	GeneratedIPList = append(GeneratedIPList, ipList)
	return ipList
}

func NewIPList(a, b, c byte) *IPList {
	ipList := &IPList{idx: 0}
	ipList.ips = make([]*IP, 127)
	ipList.idx = rand.Uint32()
	realc := c / 2
	prefix := 0
	if c%2 == 1 {
		prefix = 127
	}

	for i := 1; i < 128; i++ {
		ipList.ips[i-1] = NewIP(a, b, realc, byte(i+prefix))
	}

	return ipList
}

func (ipList *IPList) String() string {
	return ipList.ips[0].String() + " ... " + ipList.ips[126].String()
}

func (ipList *IPList) All() []*IP {
	return ipList.ips
}

func (ipList *IPList) Get() *IP {
	idx := int(ipList.idx)
	return ipList.ips[idx%len(ipList.ips)]
}

func (ipList *IPList) Next() *IP {
	forward := 1 + (rand.Uint32() % 254)
	atomic.AddUint32(&ipList.idx, forward)
	return ipList.Get()
}

func (ipList *IPList) IsAlmostBlacklisted() bool {
	blacklisted := 0
	notBlacklisted := 0

	for _, ip := range ipList.ips {
		if ip.IsBlacklisted() {
			blacklisted++
		} else {
			notBlacklisted++
		}
	}

	return blacklisted >= notBlacklisted
}
