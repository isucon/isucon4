package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

func GetMD5(data []byte) string {
	return fmt.Sprintf("%x", md5.Sum(data))
}

func GetMD5ByIO(r io.Reader) string {
	bytes, _ := ioutil.ReadAll(r)
	return GetMD5(bytes)
}

func NewFileUploadRequest(uri string, params map[string]string, paramName, path string) (*http.Request, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile(paramName, filepath.Base(path))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)

	for key, val := range params {
		_ = writer.WriteField(key, val)
	}
	writer.WriteField("type", "video/mp4")

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", uri, body)
	if err == nil {
		req.Header.Add("Content-Type", writer.FormDataContentType())
	}

	return req, err
}
