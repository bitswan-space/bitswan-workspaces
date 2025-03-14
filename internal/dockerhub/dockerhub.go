package dockerhub

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
)

// Get the latest version of the bitswan-gitops image by looking it up on dockerhub
func GetLatestDockerHubVersion(url string) (string, error) {
	results, err := getResultsField(url)
	if err != nil {
		return "latest", err
	}

	pattern := `^\d{4}-\d+-git-[a-fA-F0-9]+$`
	// Compile the regex pattern once, before the loop
	re := regexp.MustCompile(pattern)

	for _, result := range results {
		tag := result.(map[string]interface{})["name"].(string)
		if re.MatchString(tag) {
			return tag, nil
		}
	}
	return "latest", errors.New("No valid version found")
}

func GetDesiredDockerHubVersion(url string, pattern string) (string, error) {
	results, err := getResultsField(url)
	if err != nil {
		return "latest", err
	}

	// Compile the regex pattern once, before the loop
	re := regexp.MustCompile(pattern)

	for _, result := range results {
		tag := result.(map[string]interface{})["name"].(string)
		if re.MatchString(tag) {
			return tag, nil
		}
	}
	return "latest", errors.New("No valid version found")
}

// Get the results field from the dockerhub API response
func getResultsField(url string) ([]interface{}, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	return data["results"].([]interface{}), nil
}
