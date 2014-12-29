package main

import (
	"encoding/json"
	"net/http"
)

type Team struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func GetTeamByApiKey(apiKey string) *Team {
	var team *Team = nil

	req, err := http.NewRequest("GET", "https://isucon4-portal.herokuapp.com/teams/me", nil)
	if err != nil {
		return nil
	}

	req.Header.Set("Authorization", "isucon "+apiKey)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil
	}

	err = json.NewDecoder(res.Body).Decode(&team)

	if err != nil {
		return nil
	}

	return team
}
