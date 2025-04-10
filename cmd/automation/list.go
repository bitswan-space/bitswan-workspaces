package automation

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/spf13/cobra"
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

type Config struct {
	ActiveWorkspace string `toml:"active_workspace"`
}

// ANSI color codes for terminal
const (
	greenDot   = "\033[32m🟢\033[0m" // Green circle emoji
	redDot     = "\033[31m🔴\033[0m" // Red circle emoji
	greenCheck = "\033[32m✅\033[0m" // Green check
	redCheck   = "\033[31m❌\033[0m" // Red check
	bold       = "\033[1m"
	reset      = "\033[0m"
	gray       = "\033[90m"
	yellow     = "\033[33m"
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
	metadata := getMetadata(workspaceName)

	fmt.Println("Fetching automations...")

	url := fmt.Sprintf("%s/automations", metadata.GitOpsURL)
	// Send the request
	resp, err := SendAutomationRequest("GET", url, metadata.GitOpsSecret)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
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
	fmt.Printf("%s%-8s %-20s %-12s %-12s %-8s %-20s %-20s%s\n", bold, "RUNNING", "NAME", "STATE", "STATUS", "ACTIVE", "DEPLOYMENT ID", "CREATED AT", reset)
	fmt.Println(gray + "--------------------------------------------------------------------------------------------------------" + reset)

	// Print each automation
	for _, a := range automations {
		runningStatus := redDot // Default to red (inactive)
		if a.State == "running" {
			runningStatus = greenDot // Change to green if active
		}

		activeStatus := redCheck // Default to red (inactive)
		if a.Active {
			activeStatus = greenCheck // Change to green if active
		}

		// Format created_at properly
		createdAtFormatted := parseTimestamp(a.CreatedAt)

		name := a.Name
		if len(name) > 20 {
			name = name[:15] + "..."
		}

		deploymentId := a.DeploymentID
		if len(a.DeploymentID) > 20 {
			deploymentId = a.DeploymentID[:15] + "..."
		}

		// Print formatted row
		fmt.Printf("%-16s %-20s %-12s %-12s %-16s %-20s %-20s\n",
			runningStatus, name, a.State, a.Status, activeStatus, deploymentId, createdAtFormatted)
		fmt.Println(gray + "--------------------------------------------------------------------------------------------------------" + reset)
	}

	// Footer info
	fmt.Println(yellow + "✔ Running containers are marked with a green dot.\n" + reset)

	return automations, nil
}
