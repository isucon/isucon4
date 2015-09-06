package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var (
	instanceType string
	instanceId   string
	amiId        string
	cpuInfo      string
)

const (
	infoEndpoint         = "http://169.254.169.254"
	ExpectedInstanceType = "m3.xlarge"
)

func getMetaData(key string) string {
	res, err := http.Get(infoEndpoint + "/latest/meta-data/" + key)
	if err != nil {
		logger.Printf("type:fail\treason:meta-data error: %v", err)
		os.Exit(1)
	}
	if res.StatusCode == http.StatusOK {
		data, _ := ioutil.ReadAll(res.Body)
		return string(data)
	}
	logger.Printf("type:fail\treason:meta-data error: %v", err)
	os.Exit(1)
	return ""
}

func getCpuInfo() string {
	b, err := ioutil.ReadFile("/proc/cpuinfo")
	if err != nil {
		return fmt.Sprintf("%v", err)
	}
	return string(b)
}

func checkInstanceMetadata() {
	if Debug || SkipMetadata {
		instanceType = "local-machine"
		instanceId = ""
		amiId = ""
		cpuInfo = "dummy"
	} else {
		instanceType = getMetaData("instance-type")
		instanceId = getMetaData("instance-id")
		amiId = getMetaData("ami-id")
		cpuInfo = getCpuInfo()
	}

	if !(Debug || SkipMetadata) && instanceType != ExpectedInstanceType {
		logger.Printf("type:fail\treason:Instance type is miss match: %s, got: %s", ExpectedInstanceType, instanceType)
		os.Exit(1)
	}
}
