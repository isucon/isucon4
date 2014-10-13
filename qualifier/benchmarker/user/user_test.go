package user

import (
	. "testing"
)

func TestNewUser(t *T) {
	user := NewUser("isucon1", "isucon1", 1)

	if user == nil {
		t.Fatal(user)
	}

	if user.Failures != 1 {
		t.Fatal(user)
	}

	if len(user.WrongPassword) != 8 {
		t.Fatal(user)
	}
}

func TestUserFailAndSuccess(t *T) {
	user := NewUser("isucon1", "isucon1", 0)

	user.Fail()
	if user.Failures != 1 {
		t.Fatal(user)
	}
	user.Success()
	if user.Failures != 0 {
		t.Fatal(user)
	}
	user.Failures = 3
	user.Success()
	if !user.IsBlacklisted() {
		t.Fatal(user)
	}
}
