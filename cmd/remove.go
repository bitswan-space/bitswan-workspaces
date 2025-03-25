package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/bitswan-space/bitswan-gitops-cli/internal/caddyapi"
	"github.com/spf13/cobra"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "remove <gitops-name>",
		Short:        "bitswan-gitops remove",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			gitopsName := args[0]
			fmt.Printf("Removing Gitops: %s...\n", gitopsName)
			err := removeGitops(gitopsName)
			if err != nil {
				fmt.Printf("Error removing gitops: %v\n", err)
				return
			}
			fmt.Printf("Gitops %s removed successfully!\n", gitopsName)
		},
	}
	return cmd
}

func removeGitops(gitopsName string) error {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"

	// 1. Ask for confirmation
	var response string
	fmt.Printf("Are you sure you want to remove Gitops %s? Type 'yes' to confirm: ", gitopsName)
	fmt.Scanln(&response)
	if response != "yes" {
		return fmt.Errorf("operation aborted by user")
	}

	// 2. Remove docker container and volume
	workspacesFolder := filepath.Join(bitswanPath, "workspaces")
	dockerComposePath := filepath.Join(workspacesFolder, gitopsName, "deployment")
	cmd := exec.Command("docker-compose", "down", "--volumes")
	cmd.Dir = dockerComposePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute: %w", err)
	}

	// 3. Remove gitops folder
	cmd = exec.Command("rm", "-r", filepath.Join(workspacesFolder, gitopsName))
	cmd.Dir = dockerComposePath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute: %w", err)
	}

	// 4. Remove unused images
	cmd = exec.Command("docker", "image", "prune", "-a")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to execute: %w", err)
	}

	// 5. Remove entries from /etc/hosts
	hostsFilePath := "/etc/hosts"
	input, err := os.ReadFile(hostsFilePath)
	if err != nil {
		return fmt.Errorf("Failed to read /etc/hosts: %w", err)
	}

	lines := strings.Split(string(input), "\n")
	for _, line := range lines {
		if !strings.Contains(line, fmt.Sprintf("%s-gitops.bitswan.local", gitopsName)) &&
			!strings.Contains(line, fmt.Sprintf("%s-editor.bitswan.local", gitopsName)) {
			lines = append(lines, line)
		}
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(hostsFilePath, []byte(output), 0644); err != nil {
		return fmt.Errorf("Failed to write to /etc/hosts: %w", err)
	}

	// 6. Remove DNS entries from caddy
	caddyapi.RemoveCaddyRecord(gitopsName)

	return nil
}
