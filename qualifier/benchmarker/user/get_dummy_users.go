package user

import (
	"math/rand"
)

func GetDummyUsers(num int) []*User {
	users := []*User{}
	prefix := rand.Intn(len(DummyUsers))

	for i := 0; i < num; i++ {
		users = append(users, DummyUsers[(i+prefix)%len(DummyUsers)])
	}

	return users
}
