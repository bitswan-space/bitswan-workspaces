package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/bitswan-space/bitswan-workspaces/internal/caddyapi"
	"github.com/spf13/cobra"
)

// Automation represents the JSON structure
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

// Docker compose only desired field
type Compose struct {
	Services map[string]struct {
		Image string `yaml:"image"`
	} `yaml:"services"`
}

// Metadata only desired field
type Metadata struct {
	GitOpsURL    string `yaml:"gitops-url"`
	GitOpsSecret string `yaml:"gitops-secret"`
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

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "remove <gitops-name>",
		Short:        "bitswan-gitops remove",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			gitopsName := args[0]
			err := removeGitops(gitopsName)
			if err != nil {
				fmt.Errorf("Error removing gitops: %w", err)
				return
			}
		},
	}
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

func checkContainerExists(imageName string) (bool, error) {
	cmd := exec.Command("docker", "ps", "-a", "--filter", "ancestor="+imageName, "--format", "{{.ID}}")

	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return false, err
	}

	// Trim space and check if the output is empty
	output := strings.TrimSpace(out.String())
	return output != "", nil
}

// Function to delete a Docker image
func deleteDockerImage(image string) error {
	cmd := exec.Command("docker", "rmi", "-f", image) // -f forces deletion
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error deleting image %s: %s", image, out.String())
	}
	fmt.Printf("Deleted image: %s\n", image)
	return nil
}

// Function for deleting entries from /etc/hosts
func deleteHostsEntry(gitopsName string) {
	hostsFilePath := "/etc/hosts"
	input, err := os.ReadFile(hostsFilePath)
	if err != nil {
		fmt.Printf(yellow+"failed to read /etc/hosts: %v\n"+reset, err)
		return
	}

	lines := strings.Split(string(input), "\n")
	var outputLines []string

	// Define the entries to be removed
	hostsEntries := []string{
		"127.0.0.1 " + gitopsName + "-gitops.bitswan.local",
		"127.0.0.1 " + gitopsName + "-editor.bitswan.local",
	}

	found := false
	for _, entry := range hostsEntries {
		if exec.Command("grep", "-wq", entry, "/etc/hosts").Run() == nil {
			found = true
			break
		}
	}

	// No entries found to remove
	if !found {
		fmt.Println(yellow + "No entries found in /etc/hosts to remove." + reset)
		return
	}

	// Filter out the lines that match the entries
	for _, line := range lines {
		shouldRemove := false
		for _, entry := range hostsEntries {
			if strings.Contains(line, entry) {
				shouldRemove = true
				break
			}
		}
		if !shouldRemove {
			outputLines = append(outputLines, line)
		}
	}

	// Write the updated content back to /etc/hosts
	output := strings.Join(outputLines, "\n")
	cmd := exec.Command("sudo", "tee", hostsFilePath)
	cmd.Stdin = strings.NewReader(output)
	cmd.Stdout = nil
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf(yellow+"failed to write to /etc/hosts with sudo: %v\n"+reset, err)
		return
	}
}

// Remove automations
func removeAutomations(automations []Automation, token, url string) {
	client := &http.Client{}
	for _, a := range automations {
		fmt.Printf("Removing automation %s...\n", a.Name)
		req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/automations/%s", url, a.DeploymentID), nil)
		if err != nil {
			fmt.Printf("Error creating request for automation %s: %v\n", a.Name, err)
			continue
		}

		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Error sending request for automation %s: %v\n", a.Name, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Error removing automation %s: %s\n", a.Name, resp.Status)
		} else {
			fmt.Printf("Automation %s removed successfully.\n", a.Name)
		}

		resp.Body.Close()
	}
}

func removeGitops(gitopsName string) error {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"
	gitopsPath := bitswanPath + "workspaces/" + gitopsName

	// 1. Ask user for confirmation
	var confirm string

	fmt.Println("Automations in this gitops will be removed and cannot be recovered.")
	fmt.Println("Fetching automations...")

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
		fmt.Println("Error creating request:", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+metadata.GitOpsSecret)

	// Create HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending request:", err)
	}
	defer resp.Body.Close()

	// Parse the response
	var automations []Automation
	body, _ := ioutil.ReadAll(resp.Body)
	err = json.Unmarshal([]byte(body), &automations)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
	}

	fmt.Println("Automations fetched successfully.")
	fmt.Println("The following automations are running in this gitops:\n")
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

	fmt.Printf("Are you sure you want to remove %s? (yes/no): \n", gitopsName)
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Remove cancelled.")
		return nil
	}

	// 2. Remove the automations from the server
	fmt.Println("Removing automations...")
	removeAutomations(automations, metadata.GitOpsSecret, metadata.GitOpsURL)
	fmt.Println("Automations removed successfully.")

	// 2. Remove docker container and volume
	fmt.Println("Removing docker containers and volumes...")
	workspacesFolder := filepath.Join(bitswanPath, "workspaces")
	dockerComposePath := filepath.Join(workspacesFolder, gitopsName, "deployment")
	projectName := gitopsName + "-site"
	cmd := exec.Command("docker", "compose", "-p", projectName, "down", "--volumes")
	cmd.Dir = dockerComposePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute: %w", err)
	}
	fmt.Println("Docker containers and volumes removed successfully.")

	// 3. Remove images used by docker-compose
	fmt.Println("Removing images used by docker-compose...")
	dockerComposeFilePath := filepath.Join(dockerComposePath, "docker-compose.yml")
	data, err = os.ReadFile(dockerComposeFilePath)
	if err != nil {
		panic(err)
	}

	var compose Compose
	err = yaml.Unmarshal(data, &compose)
	if err != nil {
		panic(err)
	}

	for _, service := range compose.Services {
		if service.Image != "" {
			exists, err := checkContainerExists(service.Image)
			if err != nil {
				fmt.Println("Error checking images in container:", err)
			}

			if !exists {
				err = deleteDockerImage(service.Image)
				if err != nil {
					fmt.Println("Error deleting image:", err)
				}
				fmt.Println("Images removed successfully.")
			} else {
				fmt.Printf("Image %s is still in use by a different container. Skipping deletion.\n", service.Image)
			}
		}
	}

	// 4. Remove the gitops folder
	fmt.Println("Removing gitops folder...")
	cmd = exec.Command("rm", "-r", gitopsName)
	cmd.Dir = workspacesFolder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute: %w", err)
	}
	fmt.Println("GitOps folder removed successfully.")

	// 5. Remove caddy files
	fmt.Println("Removing caddy files...")
	err = caddyapi.DeleteCaddyRecords(gitopsName)
	if err != nil {
		return fmt.Errorf("Error removing caddy files: %w", err)
	}
	fmt.Println("Caddy files removed successfully.")

	// 6. Remove entries from /etc/hosts
	fmt.Println("Removing entries from /etc/hosts...")
	deleteHostsEntry(gitopsName)
	fmt.Println("Entries removed from /etc/hosts successfully.")

	return nil

}
