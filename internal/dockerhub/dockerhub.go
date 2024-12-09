package dockerhub

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
)

func GetLatestDockerHubVersion(url string) (string, error) {
	// Get the latest version of the bitswan-gitops image by looking it up on dockerhub
	resp, err := http.Get(url)
	if err != nil {
		return "latest", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "latest", err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "latest", err
	}
	results := data["results"].([]interface{})
	pattern := `^\d{4}-\d+-git-[a-fA-F0-9]+$`
	for _, result := range results {
		tag := result.(map[string]interface{})["name"].(string)
		if match, _ := regexp.MatchString(pattern, tag); match {
			return tag, nil
		}
	}
	return "latest", errors.New("No valid version found")
}