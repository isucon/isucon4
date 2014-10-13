package user

import (
	"github.com/isucon/isucon4/qualifier/benchmarker/ip"
	"sync"
	"sync/atomic"
	"time"
)

type User struct {
	*sync.Mutex
	Name          string
	RightPassword string
	WrongPassword string
	Failures      uint32
	NowInUse      bool
	MayIncomplete bool

	LastLoginedIP   *ip.IP
	LastLoginedTime time.Time
}

func NewUser(name, password string, failures uint32) *User {
	return &User{
		Mutex:         new(sync.Mutex),
		Name:          name,
		RightPassword: password,
		WrongPassword: randomString(len(password) + 1),
		Failures:      failures,
		MayIncomplete: false,
		NowInUse:      false,
	}
}

func (u *User) Fail() {
	u.Lock()
	atomic.AddUint32(&u.Failures, 1)
	u.Unlock()
}

func (u *User) Success() {
	if u.IsBlacklisted() {
		return
	}

	u.Lock()
	atomic.StoreUint32(&u.Failures, 0)
	u.Unlock()
}

func (u *User) IsBlacklisted() bool {
	u.Lock()
	defer u.Unlock()
	return atomic.LoadUint32(&u.Failures) >= 3
}

func (u *User) InUse() bool {
	u.Lock()
	defer u.Unlock()
	return u.NowInUse
}

func (u *User) Start() {
	u.Lock()
	u.NowInUse = true
	u.Unlock()
}

func (u *User) Finish() {
	u.Lock()
	u.NowInUse = false
	u.Unlock()
}

func (u *User) FlagIncomplete() {
	u.Lock()
	u.MayIncomplete = true
	u.Unlock()
}

func (u *User) IsIncomplete() bool {
	u.Lock()
	defer u.Unlock()
	return u.MayIncomplete
}
