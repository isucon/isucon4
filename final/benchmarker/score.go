package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const (
	ScoreImpression = 0.001 // 1 imp =   0.001 点
	ScoreClick      = 1.000 // 1 click = 1.000 点
	ScorePostAd     = 3.000 // 1 ad    - 50.00 点(1度以上表示されたもののみ加算)
	BonusEquality   = 5.000 // スロット内のすべての広告が表示されていれば imp スコアの n 倍を加算
	MaximumErrRate  = 25
)

func (br *BenchmarkRecipe) Score() (float64, float64, float64, interface{}) {
	var (
		success = float64(0)
		fail    = float64(0)
	)

	for _, adr := range br.advertisers {
		for _, slot := range adr.Slots {
			shownAds := 0
			totalImps := int64(0)

			for _, ad := range slot.Ads {
				totalImps += ad.Impression

				success += float64(ad.Impression) * ScoreImpression

				if ad.Impression > 0 {
					success += ScorePostAd
					shownAds++
				}
				success += float64(len(ad.ClickedUsers)) * ScoreClick
			}

			if shownAds == len(slot.Ads) {
				success += float64(totalImps) * BonusEquality
			}
		}
	}

	disqualification := false

	errCount := 0
	errErrCount := 0
	errReport := br.ErrorReport()
	for _, err := range errReport {
		errCount++
		switch err.Level {
		case ErrFatal:
			disqualification = true
		case ErrError:
			errErrCount++
			fail += ScoreClick
		case ErrNotice:
			fail += ScoreImpression
		}
	}

	if errCount > 0 && errErrCount > 0 && (errCount/errErrCount) >= MaximumErrRate {
		disqualification = true
	}

	if disqualification {
		success = 0
	}

	fail = float64(int(fail*100)) / 100
	success = float64(int(success*100)) / 100
	total := success - fail
	total = float64(int(total*100)) / 100

	data := map[string]interface{}{}

	data["errors"] = errReport.ToJSON()
	data["score"] = map[string]float64{
		"fail":    fail,
		"success": success,
		"total":   total,
	}

	return total, success, fail, data
}

func SendScore(apiKey string, total, success, fail float64) error {
	if total < 1 {
		return errors.New("0点のためスコアは送信されません")
	}

	score := map[string]float64{
		"score":     total,
		"successes": success,
		"fails":     fail,
	}

	blob, err := json.Marshal(score)
	if err != nil {
		return err
	}

	jsonBody := bytes.NewReader(blob)

	req, err := http.NewRequest("POST", "https://isucon4-portal.herokuapp.com/results", jsonBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-KEY", ApiKey)
	if MasterAPIKey != "None" {
		req.Header.Set("X-Force-Admin-Benchmark", MasterAPIKey)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	res.Body.Close()

	if res.StatusCode != http.StatusCreated {
		return errors.New(fmt.Sprintf("想定外のレスポンスコードです: %d", res.StatusCode))
	}

	return nil
}
