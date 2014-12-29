package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/go-martini/martini"
	"github.com/go-redis/redis"
	"github.com/martini-contrib/render"
)

type Ad struct {
	Slot        string `json:"slot"`
	Id          string `json:"id"`
	Title       string `json:"title"`
	Type        string `json:"type"`
	Advertiser  string `json:"advertiser"`
	Destination string `json:"destination"`
	Impressions int    `json:"impressions"`
}

type AdWithEndpoints struct {
	Ad
	Asset    string `json:"asset"`
	Redirect string `json:"redirect"`
	Counter  string `json:"counter"`
}

type ClickLog struct {
	AdId   string `json:"ad_id"`
	User   string `json:"user"`
	Agent  string `json:"agent"`
	Gender string `json:"gender"`
	Age    int    `json:"age"`
}

type Report struct {
	Ad          *Ad              `json:"ad"`
	Clicks      int              `json:"clicks"`
	Impressions int              `json:"impressions"`
	Breakdown   *BreakdownReport `json:"breakdown,omitempty"`
}

type BreakdownReport struct {
	Gender      map[string]int `json:"gender"`
	Agents      map[string]int `json:"agents"`
	Generations map[string]int `json:"generations"`
}

var rd *redis.Client

func init() {
	rd = redis.NewTCPClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   0,
	})
}

func getDir(name string) string {
	base_dir := "/tmp/go/"
	path := base_dir + name
	os.MkdirAll(path, 0755)
	return path
}

func urlFor(req *http.Request, path string) string {
	host := req.Host
	if host != "" {
		return "http://" + host + path
	} else {
		return path
	}
}

func fetch(hash map[string]string, key string, defaultValue string) string {
	if hash[key] == "" {
		return defaultValue
	} else {
		return hash[key]
	}
}

func incr_map(dict *map[string]int, key string) {
	_, exists := (*dict)[key]
	if !exists {
		(*dict)[key] = 0
	}
	(*dict)[key]++
}

func advertiserId(req *http.Request) string {
	return req.Header.Get("X-Advertiser-Id")
}

func adKey(slot string, id string) string {
	return "isu4:ad:" + slot + "-" + id
}

func assetKey(slot string, id string) string {
	return "isu4:asset:" + slot + "-" + id
}

func advertiserKey(id string) string {
	return "isu4:advertiser:" + id
}

func slotKey(slot string) string {
	return "isu4:slot:" + slot
}

func nextAdId() string {
	id, _ := rd.Incr("isu4:ad-next").Result()
	return strconv.FormatInt(id, 10)
}

func nextAd(req *http.Request, slot string) *AdWithEndpoints {
	key := slotKey(slot)
	id, _ := rd.RPopLPush(key, key).Result()
	if id == "" {
		return nil
	}
	ad := getAd(req, slot, id)
	if ad != nil {
		return ad
	} else {
		rd.LRem(key, 0, id).Result()
		return nextAd(req, slot)
	}
}

func getAd(req *http.Request, slot string, id string) *AdWithEndpoints {
	key := adKey(slot, id)
	m, _ := rd.HGetAllMap(key).Result()

	if m == nil {
		return nil
	}
	if _, exists := m["id"]; !exists {
		return nil
	}

	imp, _ := strconv.Atoi(m["impressions"])
	path_base := "/slots/" + slot + "/ads/" + id
	var ad *AdWithEndpoints
	ad = &AdWithEndpoints{
		Ad{
			m["slot"],
			m["id"],
			m["title"],
			m["type"],
			m["advertiser"],
			m["destination"],
			imp,
		},
		urlFor(req, path_base+"/asset"),
		urlFor(req, path_base+"/redirect"),
		urlFor(req, path_base+"/count"),
	}
	return ad
}

func decodeUserKey(id string) (string, int) {
	if id == "" {
		return "unknown", -1
	}
	splitted := strings.Split(id, "/")
	gender := "male"
	if splitted[0] == "0" {
		gender = "female"
	}
	age, _ := strconv.Atoi(splitted[1])

	return gender, age
}

func getLogPath(advrId string) string {
	dir := getDir("log")
	splitted := strings.Split(advrId, "/")
	return dir + "/" + splitted[len(splitted)-1]
}

func getLog(id string) map[string][]ClickLog {
	path := getLogPath(id)
	result := map[string][]ClickLog{}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return result
	}

	f, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_SH)
	if err != nil {
		panic(err)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimRight(line, "\n")
		sp := strings.Split(line, "\t")
		ad_id := sp[0]
		user := sp[1]
		agent := sp[2]
		if agent == "" {
			agent = "unknown"
		}
		gender, age := decodeUserKey(sp[1])
		if result[ad_id] == nil {
			result[ad_id] = []ClickLog{}
		}
		data := ClickLog{ad_id, user, agent, gender, age}
		result[ad_id] = append(result[ad_id], data)
	}

	return result
}

func routePostAd(r render.Render, req *http.Request, params martini.Params) {
	slot := params["slot"]

	advrId := advertiserId(req)
	if advrId == "" {
		r.Status(404)
		return
	}

	req.ParseMultipartForm(100000)
	asset := req.MultipartForm.File["asset"][0]
	id := nextAdId()
	key := adKey(slot, id)

	content_type := ""
	if len(req.Form["type"]) > 0 {
		content_type = req.Form["type"][0]
	}
	if content_type == "" && len(asset.Header["Content-Type"]) > 0 {
		content_type = asset.Header["Content-Type"][0]
	}
	if content_type == "" {
		content_type = "video/mp4"
	}

	title := ""
	if a := req.Form["title"]; a != nil {
		title = a[0]
	}
	destination := ""
	if a := req.Form["destination"]; a != nil {
		destination = a[0]
	}

	rd.HMSet(key,
		"slot", slot,
		"id", id,
		"title", title,
		"type", content_type,
		"advertiser", advrId,
		"destination", destination,
		"impressions", "0",
	)

	f, _ := asset.Open()
	defer f.Close()
	buf := bytes.NewBuffer(nil)
	io.Copy(buf, f)
	asset_data := string(buf.Bytes())

	rd.Set(assetKey(slot, id), asset_data)
	rd.RPush(slotKey(slot), id)
	rd.SAdd(advertiserKey(advrId), key)

	r.JSON(200, getAd(req, slot, id))
}

func routeGetAd(r render.Render, req *http.Request, params martini.Params) {
	slot := params["slot"]
	ad := nextAd(req, slot)
	if ad != nil {
		r.Redirect("/slots/" + slot + "/ads/" + ad.Id)
	} else {
		r.JSON(404, map[string]string{"error": "not_found"})
	}
}

func routeGetAdWithId(r render.Render, req *http.Request, params martini.Params) {
	slot := params["slot"]
	id := params["id"]
	ad := getAd(req, slot, id)
	if ad != nil {
		r.JSON(200, ad)
	} else {
		r.JSON(404, map[string]string{"error": "not_found"})
	}
}

func routeGetAdAsset(r render.Render, res http.ResponseWriter, req *http.Request, params martini.Params) {
	slot := params["slot"]
	id := params["id"]
	ad := getAd(req, slot, id)
	if ad == nil {
		r.JSON(404, map[string]string{"error": "not_found"})
		return
	}
	content_type := "application/octet-stream"
	if ad.Type != "" {
		content_type = ad.Type
	}

	res.Header().Set("Content-Type", content_type)
	data, _ := rd.Get(assetKey(slot, id)).Result()

	range_str := req.Header.Get("Range")
	if range_str == "" {
		r.Data(200, []byte(data))
		return
	}

	re := regexp.MustCompile("^bytes=(\\d*)-(\\d*)$")
	m := re.FindAllStringSubmatch(range_str, -1)

	if m == nil {
		r.Status(416)
		return
	}

	head_str := m[0][1]
	tail_str := m[0][2]

	if head_str == "" && tail_str == "" {
		r.Status(416)
		return
	}

	head := 0
	tail := 0

	if head_str != "" {
		head, _ = strconv.Atoi(head_str)
	}
	if tail_str != "" {
		tail, _ = strconv.Atoi(tail_str)
	} else {
		tail = len(data) - 1
	}

	if head < 0 || head >= len(data) || tail < 0 {
		r.Status(416)
		return
	}

	range_data := data[head:(tail + 1)]
	content_range := fmt.Sprintf("bytes %d-%d/%d", head, tail, len(data))
	res.Header().Set("Content-Range", content_range)
	res.Header().Set("Content-Length", strconv.Itoa(len(range_data)))

	r.Data(206, []byte(range_data))
}

func routeGetAdCount(r render.Render, params martini.Params) {
	slot := params["slot"]
	id := params["id"]
	key := adKey(slot, id)

	exists, _ := rd.Exists(key).Result()
	if !exists {
		r.JSON(404, map[string]string{"error": "not_found"})
		return
	}

	rd.HIncrBy(key, "impressions", 1).Result()
	r.Status(204)
}

func routeGetAdRedirect(req *http.Request, r render.Render, params martini.Params) {
	slot := params["slot"]
	id := params["id"]
	ad := getAd(req, slot, id)

	if ad == nil {
		r.JSON(404, map[string]string{"error": "not_found"})
		return
	}

	isuad := ""
	cookie, err := req.Cookie("isuad")
	if err != nil {
		if err != http.ErrNoCookie {
			panic(err)
		}
	} else {
		isuad = cookie.Value
	}
	ua := req.Header.Get("User-Agent")

	path := getLogPath(ad.Advertiser)

	var f *os.File
	f, err = os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}

	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX)
	if err != nil {
		panic(err)
	}

	fmt.Fprintf(f, "%s\t%s\t%s\n", ad.Id, isuad, ua)
	f.Close()

	r.Redirect(ad.Destination)
}

func routeGetReport(req *http.Request, r render.Render) {
	advrId := advertiserId(req)

	if advrId == "" {
		r.Status(401)
		return
	}

	report := map[string]*Report{}
	adKeys, _ := rd.SMembers(advertiserKey(advrId)).Result()
	for _, adKey := range adKeys {
		ad, _ := rd.HGetAllMap(adKey).Result()
		if ad == nil {
			continue
		}

		imp, _ := strconv.Atoi(ad["impressions"])
		data := &Report{
			&Ad{
				ad["slot"],
				ad["id"],
				ad["title"],
				ad["type"],
				ad["advertiser"],
				ad["destination"],
				imp,
			},
			0,
			imp,
			nil,
		}
		report[ad["id"]] = data
	}

	for adId, clicks := range getLog(advrId) {
		if _, exists := report[adId]; !exists {
			report[adId] = &Report{}
		}
		report[adId].Clicks = len(clicks)
	}
	r.JSON(200, report)
}

func routeGetFinalReport(req *http.Request, r render.Render) {
	advrId := advertiserId(req)

	if advrId == "" {
		r.Status(401)
		return
	}

	reports := map[string]*Report{}
	adKeys, _ := rd.SMembers(advertiserKey(advrId)).Result()
	for _, adKey := range adKeys {
		ad, _ := rd.HGetAllMap(adKey).Result()
		if ad == nil {
			continue
		}

		imp, _ := strconv.Atoi(ad["impressions"])
		data := &Report{
			&Ad{
				ad["slot"],
				ad["id"],
				ad["title"],
				ad["type"],
				ad["advertiser"],
				ad["destination"],
				imp,
			},
			0,
			imp,
			nil,
		}
		reports[ad["id"]] = data
	}

	logs := getLog(advrId)

	for adId, report := range reports {
		log, exists := logs[adId]
		if exists {
			report.Clicks = len(log)
		}

		breakdown := &BreakdownReport{
			map[string]int{},
			map[string]int{},
			map[string]int{},
		}
		for i := range log {
			click := log[i]
			incr_map(&breakdown.Gender, click.Gender)
			incr_map(&breakdown.Agents, click.Agent)
			generation := "unknown"
			if click.Age != -1 {
				generation = strconv.Itoa(click.Age / 10)
			}
			incr_map(&breakdown.Generations, generation)
		}
		report.Breakdown = breakdown
		reports[adId] = report
	}

	r.JSON(200, reports)
}

func routePostInitialize() (int, string) {
	keys, _ := rd.Keys("isu4:*").Result()
	for i := range keys {
		key := keys[i]
		rd.Del(key)
	}
	path := getDir("log")
	os.RemoveAll(path)

	return 200, "OK"
}

func main() {
	m := martini.Classic()

	m.Use(martini.Static("../public"))
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Group("/slots/:slot", func(r martini.Router) {
		m.Post("/ads", routePostAd)
		m.Get("/ad", routeGetAd)
		m.Get("/ads/:id", routeGetAdWithId)
		m.Get("/ads/:id/asset", routeGetAdAsset)
		m.Post("/ads/:id/count", routeGetAdCount)
		m.Get("/ads/:id/redirect", routeGetAdRedirect)
	})
	m.Group("/me", func(r martini.Router) {
		m.Get("/report", routeGetReport)
		m.Get("/final_report", routeGetFinalReport)
	})
	m.Post("/initialize", routePostInitialize)
	http.ListenAndServe(":8080", m)
}
