package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bitswan-space/bitswan-workspaces/cmd/automation"
	"github.com/bitswan-space/bitswan-workspaces/internal/caddyapi"
	"github.com/spf13/cobra"
)

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
	reset  = "\033[0m"
	yellow = "\033[33m"
)

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "remove <workspace-name>",
		Short:        "bitswan workspace remove",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName := args[0]
			err := removeGitops(workspaceName)
			if err != nil {
				return fmt.Errorf("error removing gitops: %w", err)
			}
			return nil
		},
	}
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
	cmd := exec.Command("docker", "rmi", image) // -f forces deletion
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
func deleteHostsEntry(workspaceName string) {
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
		"127.0.0.1 " + workspaceName + "-gitops.bitswan.local",
		"127.0.0.1 " + workspaceName + "-editor.bitswan.local",
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

func removeGitops(workspaceName string) error {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"
	gitopsPath := bitswanPath + "workspaces/" + workspaceName

	// 1. Ask user for confirmation
	var confirm string

	fmt.Println("Automations in this gitops will be removed and cannot be recovered.")

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
	automations, err := automation.GetListAutomations(workspaceName)
	if err != nil {
		fmt.Println("Error:", err)
	}

	fmt.Printf("Are you sure you want to remove %s? (yes/no): \n", workspaceName)
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Remove cancelled.")
		return nil
	}

	// 2. Remove the automations from the server
	fmt.Println("Removing automations...")
	for _, a := range automations {
		err := automation.RemoveAutomation(workspaceName, a.DeploymentID)
		if err != nil {
			return fmt.Errorf("error removing automation %s: %w", a.Name, err)
		}
	}
	fmt.Println("Automations removed successfully.")

	// 2. Remove docker container and volume
	fmt.Println("Removing docker containers and volumes...")
	workspacesFolder := filepath.Join(bitswanPath, "workspaces")
	dockerComposePath := filepath.Join(workspacesFolder, workspaceName, "deployment")
	projectName := workspaceName + "-site"
	cmd := exec.Command("docker", "compose", "-p", projectName, "down", "--volumes")
	cmd.Dir = dockerComposePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
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
	cmd = exec.Command("rm", "-r", workspaceName)
	cmd.Dir = workspacesFolder
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute: %w", err)
	}
	fmt.Println("GitOps folder removed successfully.")

	// 5. Remove caddy files
	fmt.Println("Removing caddy files...")
	err = caddyapi.DeleteCaddyRecords(workspaceName)
	if err != nil {
		return fmt.Errorf("error removing caddy files: %w", err)
	}
	fmt.Println("Caddy files removed successfully.")

	// 6. Remove entries from /etc/hosts
	fmt.Println("Removing entries from /etc/hosts...")
	deleteHostsEntry(workspaceName)
	fmt.Println("Entries removed from /etc/hosts successfully.")

	return nil

}
