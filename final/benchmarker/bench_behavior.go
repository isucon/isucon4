package main

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"time"
)

func (w *Worker) WorkAdvertiser() {
	if len(w.Slot.Ads) == len(w.Slot.Assets) {
		w.allAds = true
		// Get Report

		req, err := w.NewRequest("GET", "/me/report", nil)
		if err != nil {
			// Error
			w.AddError(NewError(ErrFatal, "/me/report", err, req))
			return
		}

		resp, report, err := w.JSONDo(req, 1*time.Minute)
		if err != nil {
			w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			// Error
			w.AddError(NewError(ErrFatal, req.URL.String(), StatusCodeMissMatch(200, resp.StatusCode), req))
			return
		}

		if result := myReportSchema.Validate(report); !result.Valid() {
			// Error
			for _, err := range result.Errors() {
				w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
			}
			return
		}

		time.Sleep(2 * time.Second)

		return
	}

	// 広告入稿

	ad := w.Slot.NewAd("")
	ad.Destination = w.DummyServer.URL + "/" + ad.Path
	postUrl := fmt.Sprintf("http://%s/slots/%s/ads", w.Host(), w.Slot.Id)

	req, err := NewFileUploadRequest(
		postUrl,
		map[string]string{
			"title":       ad.Title,
			"destination": ad.Destination,
		},
		"asset",
		ad.Asset.Path,
	)
	if err != nil {
		w.AddError(NewError(ErrFatal, postUrl, err, req))
		return
	}

	resp, value, err := w.JSONDo(req, 1*time.Minute)

	if err != nil {
		w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		w.AddError(NewError(ErrFatal, req.URL.String(), StatusCodeMissMatch(200, resp.StatusCode), req))
		return
	}

	if result := adSchema.Validate(value); !result.Valid() {
		for _, err := range result.Errors() {
			w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
		}
		return
	}

	if id, ok := value["id"]; ok {
		if ids, ok := id.(string); ok {
			ad.Id = ids
		}
	}

	w.Slot.Lock()
	if ad.Id != "" {
		w.Slot.idAndAd[ad.Id] = ad
		w.Slot.pathAndAd[ad.Path] = ad
	}
	w.Slot.Ads = append(w.Slot.Ads, ad)
	w.Slot.Unlock()
}

func (w *Worker) ValidateReport() {
	if w.Advertiser.Validated {
		return
	}

	w.Advertiser.Validated = true

	allAds := w.Advertiser.AllAds()
	idMap := map[string]*Ad{}
	for _, ad := range allAds {
		idMap[ad.Id] = ad
	}

	req, err := w.NewRequest("GET", "/me/report", nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrFatal, "/me/report", err, req))
		return
	}

	resp, report, err := w.JSONDo(req, 3*time.Minute)
	if err != nil {
		w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Error
		w.AddError(NewError(ErrFatal, req.URL.String(), StatusCodeMissMatch(200, resp.StatusCode), req))
		return
	}

	if result := myReportSchema.Validate(report); !result.Valid() {
		// Error
		for _, err := range result.Errors() {
			w.AddError(NewError(ErrFatal, req.URL.String(), err, req))
		}
		return
	}

	freq, err := w.NewRequest("GET", "/me/final_report", nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrFatal, "/me/final_report", err, freq))
		return
	}

	fresp, freport, err := w.JSONDo(freq, 3*time.Minute)
	if err != nil {
		// Error
		w.AddError(NewError(ErrFatal, freq.URL.String(), err, freq))
		return
	}
	defer resp.Body.Close()

	if fresp.StatusCode != 200 {
		// Error
		w.AddError(NewError(ErrFatal, freq.URL.String(), StatusCodeMissMatch(200, fresp.StatusCode), freq))
		return
	}

	if result := finalReportSchema.Validate(freport); !result.Valid() {
		// Error
		for _, err := range result.Errors() {
			w.AddError(NewError(ErrFatal, freq.URL.String(), err, freq))
		}
		return
	}

	adsCnt := 0

	for id, val := range report {
		data, ok := val.(map[string]interface{})

		if !ok {
			w.AddError(NewError(ErrFatal, req.URL.String(), errors.New("JSON のスキーマが壊れています"), req))
			return
		}

		actAd, ok := idMap[id]
		if !ok {
			w.AddError(NewError(ErrFatal, req.URL.String(), errors.New("存在しないはずのIDの広告が報告されています"), req))
		}

		impressions, ok := data["impressions"].(float64)
		if ok && int64(impressions) < actAd.Impression {
			w.AddError(NewError(ErrFatal, req.URL.String(), errors.New("ID "+id+" のインプレッション数が不正です"), req))
		}

		clicks, ok := data["clicks"].(float64)
		if ok && int(clicks) < len(actAd.ClickedUsers) {
			w.AddError(NewError(ErrFatal, req.URL.String(), errors.New("ID "+id+" のクリック数が不正です"), req))
		}
		adsCnt++
	}

	if len(allAds) != adsCnt {
		w.AddError(NewError(ErrFatal, req.URL.String(), errors.New("報告されるべき広告の数が一致していません"), req))
	}

	adsCnt = 0

	for id, val := range freport {
		data := val.(map[string]interface{})

		actAd, ok := idMap[id]
		if !ok {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("存在しないはずのIDの広告が報告されています"), freq))
			return
		}

		impressions, ok := data["impressions"].(float64)
		if ok && int64(impressions) < actAd.Impression {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("ID "+id+" のインプレッション数が不正です"), freq))
			return
		}

		clicks, ok := data["clicks"].(float64)
		if ok && int(clicks) < len(actAd.ClickedUsers) {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("ID "+id+" のクリック数が不正です"), freq))
			return
		}

		agents, gender, generations := actAd.BreakDown()

		dAgents := map[string]float64{}
		dGender := map[string]float64{}
		dGenerations := map[string]float64{}

		dataBrekdown, ok := data["breakdown"].(map[string]interface{})
		if !ok {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("breakdownの型が不正です"), freq))
			return
		}
		ddAgents, ok := dataBrekdown["agents"].(map[string]interface{})
		if !ok {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("agentsの型が不正です"), freq))
			return
		}
		ddGender, ok := dataBrekdown["gender"].(map[string]interface{})
		if !ok {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("genderの型が不正です"), freq))
			return
		}
		ddGenerations, ok := dataBrekdown["generations"].(map[string]interface{})
		if !ok {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("generationsの型が不正です"), freq))
			return
		}

		for k, v := range ddAgents {
			dAgents[k], ok = v.(float64)
		}

		for k, v := range ddGender {
			dGender[k], ok = v.(float64)
		}

		for k, v := range ddGenerations {
			dGenerations[k], ok = v.(float64)
		}

		if !reflect.DeepEqual(agents, dAgents) {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("ID "+id+" の UA 統計情報が不正です"), freq))
		}

		if !reflect.DeepEqual(gender, dGender) {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("ID "+id+" の性別統計情報が不正です"), freq))
		}

		if !reflect.DeepEqual(generations, dGenerations) {
			w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("ID "+id+" の世代統計情報が不正です"), freq))
		}

		adsCnt++
	}

	if len(allAds) != adsCnt {
		w.AddError(NewError(ErrFatal, freq.URL.String(), errors.New("報告されるべき広告の数が一致していません"), freq))
	}
}

func (w *Worker) WorkUser() {
	w.Slot.Lock()
	if len(w.Slot.Ads) != len(w.Slot.Assets) {
		w.Slot.Unlock()
		return
	}
	w.Slot.Unlock()

	// ランダムなユーザーをアサイン
	w.User = GetRandomUser()

	// Ad 取ってくる
	adUrl := "/slots/" + w.Slot.Id + "/ad"
	req, err := w.NewRequest("GET", adUrl, nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, adUrl, err, req))
		return
	}

	resp, ad, err := w.JSONDo(req, 3*time.Minute)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, req.URL.String(), err, req))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// Error
		w.AddError(NewError(ErrError, req.URL.String(), StatusCodeMissMatch(200, resp.StatusCode), req))
		return
	}

	if result := adSchema.Validate(ad); !result.Valid() {
		for _, err := range result.Errors() {
			w.AddError(NewError(ErrError, req.URL.String(), err, req))
		}
		return
	}

	adId := ""
	if aid, ok := ad["id"]; ok {
		if asid, ok := aid.(string); ok {
			adId = asid
		}
	}

	w.Slot.Lock()
	actAd := w.Slot.idAndAd[adId]
	w.Slot.Unlock()

	assetUrl := ad["asset"].(string)

	// 動画見たいマン
	asreq, err := http.NewRequest("GET", assetUrl, nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, assetUrl, err, asreq))
		return
	}

	asres, err := w.Do(asreq, 1*time.Minute)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, asreq.URL.String(), err, asreq))
		return
	}
	defer asres.Body.Close()

	if asres.StatusCode != 200 {
		w.AddError(NewError(ErrError, asreq.URL.String(), StatusCodeMissMatch(200, asres.StatusCode), asreq))
		return
	}

	asMD5 := ""
	if asres.Header.Get(CachedHeader) == CachedHeaderVal {
		asMD5 = asres.Header.Get(CachedMD5Header)
	} else {
		asMD5 = GetMD5ByIO(asres.Body)
	}

	if asMD5 != actAd.Asset.MD5 {
		// Error
		w.AddError(NewError(ErrError, asreq.URL.String(), MD5MissMatch(actAd.Asset.MD5, asMD5), asreq))
		return
	}

	counterUrl := ad["counter"].(string)

	// インプレッション追加するマン
	creq, err := w.NewRequest("POST", counterUrl, nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, counterUrl, err, creq))
		return
	}

	cres, err := w.Do(creq, 1*time.Minute)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, counterUrl, err, creq))
		return
	}
	defer cres.Body.Close()

	if cres.StatusCode != 204 {
		w.AddError(NewError(ErrError, counterUrl, StatusCodeMissMatch(204, cres.StatusCode), creq))
		return
	}

	actAd.IncrImp()

	redirectUrl := ad["redirect"].(string)

	// なんと CTR 100% ！！！！！！！！！！
	// 入稿される動画広告には電子ドラッグ的サブリミナル効果が入ってるという設定です
	ireq, err := http.NewRequest("GET", redirectUrl, nil)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, redirectUrl, err, ireq))
		return
	}

	ires, err := w.Do(ireq, 1*time.Minute)
	if err != nil {
		// Error
		w.AddError(NewError(ErrError, redirectUrl, err, ireq))
		return
	}

	if ires.StatusCode != 204 || ires.Header.Get(ValidationHeaderKey) != ValidationHeaderVal {
		// Error
		w.AddError(NewError(ErrError, redirectUrl, "Redirect not operated", ireq))
		return
	}

	actAd.Click(w.User)
}
