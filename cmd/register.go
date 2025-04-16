package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type DeviceAuthorizationResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type TokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	RefreshToken     string `json:"refresh_token"`
	TokenType        string `json:"token_type"`
	NotBeforePolicy  int    `json:"not-before-policy"`
	SessionState     string `json:"session_state"`
	Scope            string `json:"scope"`
}

type AutomationServerYaml struct {
	AocBeURL    string `yaml:"aoc_be_url"`
	EmqxURL     string `yaml:"emqx_url"`
	AccessToken string `yaml:"access_token"`
}

func newRegisterCmd() *cobra.Command {
	var serverName string
	var aocUrl string
	intervalSeconds := 5

	cmd := &cobra.Command{
		Use:          "register",
		Short:        "Register workspace as automation server",
		Args:         cobra.NoArgs,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := sendRequest("POST", fmt.Sprintf("http://%s:8000/api/cli/register/", aocUrl))
			if err != nil {
				return fmt.Errorf("error sending request: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("failed to register workspace: %s", resp.Status)
			}

			var deviceAuthorizationResponse DeviceAuthorizationResponse
			body, _ := ioutil.ReadAll(resp.Body)
			err = json.Unmarshal([]byte(body), &deviceAuthorizationResponse)
			if err != nil {
				return fmt.Errorf("error decoding JSON: %w", err)
			}

			localHost := "localhost:8080" // Replace with your desired host (include port if needed)

			updatedVerificationURIComplete, err := url.Parse(deviceAuthorizationResponse.VerificationURIComplete)
			if err != nil {
				log.Fatal(err)
			}

			updatedVerificationURIComplete.Host = localHost

			fmt.Printf("Please visit the following URL to authorize the device:\n%s\n", updatedVerificationURIComplete.String())

			for {
				resp, err = sendRequest("GET", fmt.Sprintf("http://%s:8000/api/cli/register?device_code=%s&server_name=%s", aocUrl, deviceAuthorizationResponse.DeviceCode, serverName))
				if err != nil {
					return fmt.Errorf("error sending request: %w", err)
				}

				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					break
				}

				// Parse error response
				var errResp map[string]interface{}
				body, _ = ioutil.ReadAll(resp.Body)
				if err := json.Unmarshal(body, &errResp); err != nil {
					return fmt.Errorf("error parsing error response: %v", err)
				}

				switch errResp["error"] {
				case "authorization_pending":
					// keep polling
				case "slow_down":
					intervalSeconds += 5
				case "expired_token", "access_denied":
					return fmt.Errorf("authorization failed: %s", errResp["error"])
				default:
					return fmt.Errorf("unexpected error: %s", errResp["error"])
				}

				// Wait before next poll
				time.Sleep(time.Duration(intervalSeconds) * time.Second)
			}

			var tokenResponse TokenResponse
			body, _ = ioutil.ReadAll(resp.Body)
			err = json.Unmarshal([]byte(body), &tokenResponse)
			if err != nil {
				return fmt.Errorf("error decoding JSON: %w", err)
			}

			saveAutomationServerYaml(aocUrl, "", tokenResponse.AccessToken)

			fmt.Printf("Successfully registered workspace as automation server. You can close the browser tab.\n")
			fmt.Println("Access token, AOC BE URL, and EMQX URL have been saved to ~/.config/bitswan/aoc/automation_server.yaml.")

			return nil
		},
	}

	cmd.Flags().StringVar(&serverName, "server-name", "", "Server name")
	cmd.Flags().StringVar(&aocUrl, "aoc", "", "Automation operation server URL")

	return cmd
}

func sendRequest(method, url string) (*http.Response, error) {
	// Create a new GET request
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Set the request headers
	req.Header.Add("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error creating client: %w", err)
	}
	return resp, nil
}

func saveAutomationServerYaml(aocBeURL string, emqxURL string, accessToken string) error {
	automationServerYaml := AutomationServerYaml{
		AocBeURL:    aocBeURL,
		EmqxURL:     emqxURL,
		AccessToken: accessToken,
	}

	aocDir := filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "aoc")

	// Marshal to YAML
	yamlData, err := yaml.Marshal(automationServerYaml)
	if err != nil {
		return fmt.Errorf("failed to marshal automation server yaml: %w", err)
	}

	// Write to file
	automationServerYamlPath := filepath.Join(aocDir, "automation_server.yaml")
	if err := os.WriteFile(automationServerYamlPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write automation server yaml file: %w", err)
	}

	return nil

}
