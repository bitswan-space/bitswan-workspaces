package oauth

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Enabled       bool
	ClientId      string
	ClientSecret  string
	IssuerUrl     string
	RedirectUrl   string
	Scopes        string
	CookieSecret  string
	EmailDomains  []string
	AllowedGroups []string
}

func GetOauthConfig(workspaceName string) *Config {
	var config Config
	fmt.Println("Getting OAuth config for workspace:", workspaceName)
	workspacePath := os.Getenv("HOME") + "/.config/bitswan/workspaces/" + workspaceName 
	configPath := workspacePath + "/secrets/oauth-config.yaml"

	if _, err := os.Stat(configPath); os.IsNotExist(err) {	
		fmt.Println("No OAuth config found, skipping...")
		return nil
	}

	fmt.Println("Config path:", configPath)

	fileContent, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Println("Error reading OAuth config file:", err)
		return nil
	}

	if err := yaml.Unmarshal(fileContent, &config); err != nil {
		fmt.Println("Error unmarshalling OAuth config file:", err)
		return nil
	}

	requiredFields := map[string]string{
		"client_id":     config.ClientId,
		"client_secret": config.ClientSecret,
		"issuer_url":    config.IssuerUrl,
		"cookie_secret": config.CookieSecret,
	}

	for field, value := range requiredFields {
		if value == "" {
			fmt.Printf("Error: %s is required\n", field)
			return nil
		}
	}

	return &config
}

