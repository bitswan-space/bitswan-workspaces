package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
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

func removeGitops(gitopsName string) error {
	// 1. Ask user for confirmation
	var confirm string
	fmt.Printf("Are you sure you want to remove %s? (yes/no): ", gitopsName)
	fmt.Scanln(&confirm)
	if confirm != "yes" {
		fmt.Println("Remove cancelled.")
		return nil
	}

	// 2. Remove the gitops from docker
	dockerComposePath := filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "workspaces", gitopsName, "deployment", "docker-compose.yml")
	cmd := exec.Command("docker", "compose", "-f", dockerComposePath, "down", "-v").Run()
	if cmd != nil {
		return fmt.Errorf("Error removing gitops from docker: %w", cmd)
	}

	// 3. Remove the gitops from the local filesystem

	return nil
}
