package ip

import (
	. "testing"
)

func TestNewIPList(t *T) {
	ipList := NewIPList(127, 0, 1)

	if len(ipList.ips) != 254 {
		t.Fatal(ipList)
	}

	t.Log(ipList)
}

func TestIPListIsAlmostBlacklisted(t *T) {
	ipList := NewIPList(127, 0, 1)

	for i := 0; i < 127; i++ {
		ip := ipList.ips[i]

		for l := 0; l < 10; l++ {
			ip.Fail()
		}
	}

	if !ipList.IsAlmostBlacklisted() {
		t.Fatal("Not blacklisted")
	}
}
