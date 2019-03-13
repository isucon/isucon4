package user

import (
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
)

var (
	DummyUsersTSVMD5     string
	DummyUsersUsedTSVMD5 string
	DebugMode            string
)

var DummyUsers []*User

func init() {
	Debug := DebugMode == "true"
	var usersTsvPath string
	var usersUsedTsvPath string

	if Debug {
		usersTsvPath = "./sql/dummy_users.tsv"
		usersUsedTsvPath = "./sql/dummy_users_used.tsv"
	} else {
		usersTsvPath = "/home/isucon/sql/dummy_users.tsv"
		usersUsedTsvPath = "/home/isucon/sql/dummy_users_used.tsv"
	}

	duMD5 := getMD5(usersTsvPath)
	if duMD5 != DummyUsersTSVMD5 {
		panic(fmt.Errorf("Broken %s", usersTsvPath))
	}

	duuMD5 := getMD5(usersUsedTsvPath)
	if duuMD5 != DummyUsersUsedTSVMD5 {
		panic(fmt.Errorf("Broken %s", usersUsedTsvPath))
	}

	DummyUsers = make([]*User, 0)
	failureMap := map[string]string{}

	fp, err := os.Open(usersUsedTsvPath)
	if err != nil {
		panic(err)
	}

	reader := csv.NewReader(fp)
	reader.Comma = '\t'
	reader.LazyQuotes = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
		failureMap[record[1]] = record[2]
	}

	fp, err = os.Open(usersTsvPath)
	if err != nil {
		panic(err)
	}

	reader = csv.NewReader(fp)
	reader.Comma = '\t'
	reader.LazyQuotes = true

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}

		failureCount := 0

		if strcount, ok := failureMap[record[1]]; ok {
			failureCount, err = strconv.Atoi(strcount)
			if err != nil {
				panic(err)
			}
		}

		DummyUsers = append(DummyUsers, NewUser(record[1], record[2], uint32(failureCount)))
	}
}

func getMD5(filename string) string {
	fp, err := os.Open(filename)
	if err != nil {
		return ""
	}

	md5hash := md5.New()
	io.Copy(md5hash, fp)
	return fmt.Sprintf("%x", md5hash.Sum(nil))
}
