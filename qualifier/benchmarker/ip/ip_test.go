package ip

import (
	. "testing"
)

func TestNewIP(t *T) {
	ip := NewIP(127, 0, 0, 1)

	t.Log(ip.IP)
}

func TestIPD(t *T) {
	ip := NewIP(127, 0, 0, 1)

	if ip.D() != 1 {
		t.Fatal(ip.D())
	}

	ip = NewIP(127, 0, 0, 255)

	if ip.D() != 255 {
		t.Fatal(ip.D())
	}
}

func TestIPIsBlackListed(t *T) {
	ip := NewIP(127, 0, 0, 1)

	ip.Failures = 9
	ip.Success()

	if ip.Failures != 0 {
		t.Fatal(ip)
	}

	for i := 0; i < 10; i++ {
		ip.Fail()
	}

	if !ip.IsBlacklisted() {
		t.Fatal(ip.Failures)
	}

	ip.Success()
	if !ip.IsBlacklisted() {
		t.Fatal(ip.Failures)
	}
}
