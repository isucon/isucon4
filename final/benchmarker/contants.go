package main

import (
	"errors"
	"github.com/rosylilly/envdef"
)

const (
	DEFAULT_WORKLOAD = 1
)

var (
	WORKER_RUNNING_DURATION  = envdef.Get("HALLEY_RUNNING_DURATION", "1m")
	WORKER_TIMEOUT_DURATION  = envdef.Get("HALLEY_TIMEOUT_DURATION", "20s")
	WORKER_ABORTING_DURATION = envdef.Get("HALLEY_ABORTING_DURATION", "20s")
)

var (
	ErrRequestTimeout  = errors.New("Connection timeout")
	ErrRequestCanceled = errors.New("Request cancelled because benchmark finished")
)

var (
	GenerateAdvertisersCount = 1000
	GenerateUsersCount       = 10000
)

var (
	UserUnderAge      = 13
	UserUpperAge      = 80
	UserDNTPercentage = 100
)

const (
	AdvertiserWorker WorkerRole = iota
	UserWorker
)

const (
	MaxAdvertisersCount = 15
)

const AssetsSparation = 5

const (
	ValidationHeaderKey = "X-CHOSEN-GREEN-TEA"
	ValidationHeaderVal = "AYATAKA"
	CachedHeader        = "X-Halley-Cached"
	CachedHeaderVal     = "true"
	CachedMD5Header     = "X-Halley-MD5"
)

var ApiKey = envdef.Get("ISUCON_API_KEY", "None")

const (
	TimeFormat = "2006-01-02 15:04:05"
)

var (
	AssetsDir    = envdef.Get("HALLEY_ASSETS_DIR", "/home/isucon/creatives")
	MasterIP     = envdef.Get("HALLEY_MASTER_HOST", "10.11.54.62")
	MasterHost   = MasterIP + ":9091"
	MasterAPIKey = envdef.Get("HALLEY_MASTER_API_KEY", "None")
)
