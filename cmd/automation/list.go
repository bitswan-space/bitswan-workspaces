package automation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Automation struct {
	ContainerID  string `json:"container_id"`
	EndpointName string `json:"endpoint_name"`
	CreatedAt    string `json:"created_at"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Status       string `json:"status"`
	DeploymentID string `json:"deployment_id"`
	Active       bool   `json:"active"`
}

// Metadata only desired field
type Metadata struct {
	GitOpsURL    string `yaml:"gitops-url"`
	GitOpsSecret string `yaml:"gitops-secret"`
}

type Config struct {
	ActiveWorkspace string `toml:"active_workspace"`
}

// ANSI color codes for terminal
const (
	green  = "\033[32m●\033[0m" // Green dot
	red    = "\033[31m●\033[0m" // Red dot
	bold   = "\033[1m"
	reset  = "\033[0m"
	gray   = "\033[90m"
	yellow = "\033[33m"
)

func newListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available bitswan workspace automations",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName, err := getWorkspaceName()
			if err != nil {
				return fmt.Errorf("failed to get active workspace from config.toml: %v", err)
			}
			_, err = GetListAutomations(workspaceName)
			if err != nil {
				return fmt.Errorf("failed to list automations: %v", err)
			}
			return nil
		},
	}

	// Add subcommands to workspace

	return cmd
}

// Parse custom timestamp format
func parseTimestamp(timestamp string) string {
	layout := "2006-01-02T15:04:05.999999"
	t, err := time.Parse(layout, timestamp)
	if err != nil {
		return "Invalid Date"
	}
	return t.Format("02 Jan 2006 15:04") // Format as "DD MMM YYYY HH:MM"
}

func getWorkspaceName() (string, error) {
	var conf Config
	configPath := filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "config.toml")

	file, err := os.Open(configPath)
	if err != nil {
		return "", fmt.Errorf("failed to open config.toml: %v", err)
	}
	defer file.Close()

	if _, err := toml.NewDecoder(file).Decode(&conf); err != nil {
		return "", fmt.Errorf("failed to decode config.toml: %v", err)
	}

	return conf.ActiveWorkspace, nil
}

func GetListAutomations(workspaceName string) ([]Automation, error) {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"
	gitopsPath := bitswanPath + "workspaces/" + workspaceName
	metadataPath := gitopsPath + "/metadata.yaml"

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		panic(err)
	}

	var metadata Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		panic(err)
	}

	// Create a new GET request
	req, err := http.NewRequest("GET", metadata.GitOpsURL+"/automations/", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+metadata.GitOpsSecret)

	// Create HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var automations []Automation
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal([]byte(body), &automations)
	if err != nil {
		return nil, fmt.Errorf("error decoding JSON: %w", err)
	}

	fmt.Println("Automations fetched successfully.")
	fmt.Print("The following automations are running in this gitops:\n\n")
	// Print table header
	fmt.Printf("%s%-8s %-20s %-12s %-10s %-20s %-20s%s\n", bold, "ACTIVE", "NAME", "STATE", "STATUS", "DEPLOYMENT ID", "CREATED AT", reset)
	fmt.Println(gray + "-----------------------------------------------------------------------------------------" + reset)

	// Print each automation
	for _, a := range automations {
		activeStatus := red // Default to red (inactive)
		if a.Active {
			activeStatus = green // Change to green if active
		}

		// Format created_at properly
		createdAtFormatted := parseTimestamp(a.CreatedAt)

		// Print formatted row
		fmt.Printf("%-17s %-20s %-12s %-10s %-20s %-20s\n",
			activeStatus, a.Name, a.State, a.Status, a.DeploymentID, createdAtFormatted)
	}

	// Footer info
	fmt.Println(gray + "-----------------------------------------------------------------------------------------" + reset)
	fmt.Println(yellow + "✔ Running containers are marked with a green dot.\n" + reset)

	return automations, nil
}
