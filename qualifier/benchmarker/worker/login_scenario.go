package worker

import (
	"github.com/isucon/isucon4/qualifier/benchmarker/ip"
	"github.com/isucon/isucon4/qualifier/benchmarker/user"
	"math/rand"
	"time"
)

var defaultExpectedAssets = map[string]string{
	"/stylesheets/bootstrap.min.css": "385b964b68acb68d23cb43a5218fade9",
	"/stylesheets/bootflat.min.css":  "6409900e808d514534adc1bd0e46bbbb",
	"/stylesheets/isucon-bank.css":   "5c457844dd5ffb85c9bcd3e3a8c182ad",
	"/images/isucon-bank.png":        "908b13678cb5bc56a4f83bbb2eb1dce6",
}

func (w *Worker) Login(from *ip.IP, user *user.User) error {
	if from.IsBlacklisted() || user.IsBlacklisted() {
		return w.LoginWithBlocked(from, user)
	}

	n := rand.Intn(10)

	if n < 6 && from.D()%3 == 0 {
		return w.LoginWithSuccess(from, user)
	} else {
		return w.LoginWithFail(from, user)
	}
}

func (w *Worker) LoginWithSuccess(from *ip.IP, user *user.User) error {
	defer w.Reset()
	defer func() {
		user.LastLoginedIP = from
		user.LastLoginedTime = time.Now()
	}()

	topPage := NewScenario("GET", "/")
	topPage.IP = from
	topPage.ExpectedStatusCode = 200
	topPage.ExpectedSelectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.ExpectedAssets = defaultExpectedAssets

	err := topPage.Play(w)

	if err != nil {
		return err
	}

	login := NewScenario("POST", "/login")
	login.IP = from
	login.PostData = map[string]string{
		"login":    user.Name,
		"password": user.RightPassword,
	}
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/mypage"
	if user.LastLoginedIP != nil {
		login.ExpectedHTML = map[string]string{
			"//*[@id='last-logined-ip']": user.LastLoginedIP.String(),
		}
		login.ExpectedLastLoginedAt = user.LastLoginedTime
	}
	login.ExpectedAssets = defaultExpectedAssets

	err = login.Play(w)

	from.Success()
	user.Success()

	return err
}

func (w *Worker) LoginWithFail(from *ip.IP, user *user.User) error {
	defer w.Reset()
	defer func() {
		from.Fail()
		user.Fail()
	}()

	topPage := NewScenario("GET", "/")
	topPage.IP = from
	topPage.ExpectedStatusCode = 200
	topPage.ExpectedSelectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.ExpectedAssets = defaultExpectedAssets

	err := topPage.Play(w)

	if err != nil {
		return err
	}

	login := NewScenario("POST", "/login")
	login.IP = from
	login.PostData = map[string]string{
		"login":    user.Name,
		"password": user.WrongPassword,
	}
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/"
	login.ExpectedHTML = map[string]string{
		"//*[@id='notice-message']": "Wrong username or password",
	}
	login.ExpectedAssets = defaultExpectedAssets

	err = login.Play(w)

	return err
}

func (w *Worker) LoginWithBlocked(from *ip.IP, user *user.User) error {
	defer w.Reset()
	defer func() {
		from.Fail()
		user.Fail()
	}()

	topPage := NewScenario("GET", "/")
	topPage.IP = from
	topPage.ExpectedStatusCode = 200
	topPage.ExpectedSelectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.ExpectedAssets = defaultExpectedAssets

	err := topPage.Play(w)

	if err != nil {
		return err
	}

	login := NewScenario("POST", "/login")
	login.IP = from
	login.PostData = map[string]string{
		"login":    user.Name,
		"password": user.RightPassword,
	}
	login.ExpectedStatusCode = 200
	login.ExpectedLocation = "/"
	login.ExpectedHTML = map[string]string{
		"//*[@id='notice-message']": "This account is locked.",
	}
	if from.IsBlacklisted() {
		login.ExpectedHTML["//*[@id='notice-message']"] = "You're banned."
	}
	login.ExpectedAssets = defaultExpectedAssets

	err = login.Play(w)

	return err
}
