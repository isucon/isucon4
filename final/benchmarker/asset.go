package main

import (
	"io/ioutil"
	"path"
)

type Asset struct {
	Path string
	MD5  string
}

var Assets []*Asset

func PreloadAssets(dir string) {
	if len(Assets) > 0 {
		return
	}

	Assets = []*Asset{}
	files, _ := ioutil.ReadDir(dir)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fp := path.Join(dir, file.Name())
		data, _ := ioutil.ReadFile(fp)
		md5 := GetMD5(data)
		asset := &Asset{fp, md5}
		Assets = append(Assets, asset)
	}
}

func GetAssets(idx int) []*Asset {
	l := len(Assets)
	max := l / AssetsSparation
	if l%AssetsSparation != 0 {
		max++
	}
	for idx >= max {
		idx -= max
	}
	stIdx := (idx * AssetsSparation)
	edIdx := stIdx + AssetsSparation
	if edIdx > l {
		edIdx = l
	}

	return Assets[stIdx:edIdx]
}
