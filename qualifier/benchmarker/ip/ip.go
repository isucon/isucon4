package ip

import (
	"net"
	"sync"
	"sync/atomic"
)

type IP struct {
	*net.IP
	*sync.Mutex
	Failures uint32
	MayIncomplete bool
}

func NewIP(a, b, c, d byte) *IP {
	ip := net.IPv4(a, b, c, d)
	return &IP{
		IP:       &ip,
		Mutex:    new(sync.Mutex),
		Failures: 0,
	}
}

func (ip *IP) D() int {
	return int(ip.IP.To4()[3])
}

func (ip *IP) String() string {
	return ip.IP.String()
}

func (ip *IP) Success() {
	if ip.IsBlacklisted() {
		return
	}

	ip.Lock()
	atomic.StoreUint32(&ip.Failures, 0)
	ip.Unlock()
}

func (ip *IP) Fail() {
	ip.Lock()
	atomic.AddUint32(&ip.Failures, 1)
	ip.Unlock()
}

func (ip *IP) IsBlacklisted() bool {
	ip.Lock()
	defer ip.Unlock()
	return atomic.LoadUint32(&ip.Failures) >= 10
}

func (ip *IP) FlagIncomplete() {
	ip.Lock()
	ip.MayIncomplete = true
	ip.Unlock()
}

func (ip *IP) IsIncomplete() bool {
	ip.Lock()
	defer ip.Unlock()
	return ip.MayIncomplete
}
