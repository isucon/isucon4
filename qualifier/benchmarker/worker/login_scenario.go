package worker

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/KentaKudo/isucon4/qualifier/benchmarker/ip"
	"github.com/KentaKudo/isucon4/qualifier/benchmarker/user"
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
	topPage.Expectation.StatusCode = http.StatusOK
	topPage.Expectation.Selectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.Expectation.Assets = defaultExpectedAssets

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
	login.Expectation.StatusCode = http.StatusOK
	login.Expectation.Location = "/mypage"
	if user.LastLoginedIP != nil {
		login.Expectation.HTML = map[string]string{"//*[@id='last-logined-ip']": user.LastLoginedIP.String()}
		login.Expectation.LastLoginedAt = user.LastLoginedTime
	}
	login.Expectation.Assets = defaultExpectedAssets

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
	topPage.Expectation.StatusCode = http.StatusOK
	topPage.Expectation.Selectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.Expectation.Assets = defaultExpectedAssets

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
	login.Expectation.StatusCode = http.StatusOK
	login.Expectation.Location = "/"
	login.Expectation.HTML = map[string]string{"//*[@id='notice-message']": "Wrong username or password"}
	login.Expectation.Assets = defaultExpectedAssets

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
	topPage.Expectation.StatusCode = http.StatusOK
	topPage.Expectation.Selectors = []string{"//input[@name='login']", "//input[@name='password']", "//*[@type='submit']"}
	topPage.Expectation.Assets = defaultExpectedAssets

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
	login.Expectation.StatusCode = http.StatusOK
	login.Expectation.Location = "/"
	login.Expectation.HTML = map[string]string{"//*[@id='notice-message']": "This account is locked."}
	if from.IsBlacklisted() {
		login.Expectation.HTML["//*[@id='notice-message']"] = "You're banned."
	}
	login.Expectation.Assets = defaultExpectedAssets

	err = login.Play(w)

	return err
}
