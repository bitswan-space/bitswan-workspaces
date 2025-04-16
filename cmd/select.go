package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func newSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <workspace>",
		Short: "Select a workspace for activation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspace := args[0]
			bitswanDir := filepath.Join(os.Getenv("HOME"), ".config", "bitswan")

			err := checkValidWorkspace(workspace, bitswanDir)
			if err != nil {
				return fmt.Errorf("error: %w", err)
			}

			// Proceed to write the active workspace to the config file
			configPath := filepath.Join(bitswanDir, "config.toml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					return fmt.Errorf("failed to create config directory: %w", err)
				}
				file, err := os.Create(configPath)
				if err != nil {
					return fmt.Errorf("failed to create config file: %w", err)
				}
				defer file.Close()
				fmt.Println("Config file created at:", configPath)
			} else if err != nil {
				return fmt.Errorf("error checking config file: %w", err)
			} else {
				fmt.Println("Config file already exists at:", configPath)
				fmt.Println("Overwriting the active workspace...")
			}

			// Write the active workspace to the config file
			configContent := fmt.Sprintf("active_workspace = \"%s\"\n", workspace)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				return fmt.Errorf("failed to write to config file: %w", err)
			}

			fmt.Printf("Active workspace set to '%s' in config file.\n", workspace)
			return nil
		},
	}
}

func checkValidWorkspace(workspace string, bitswanDir string) error {
	workspacesDir := filepath.Join(bitswanDir, "workspaces")

	// Check if the workspaces directory exists
	if _, err := os.Stat(workspacesDir); os.IsNotExist(err) {
		return fmt.Errorf("workspaces directory does not exist: %s", workspacesDir)
	}

	// Validate if the provided workspace exists in the workspaces directory
	workspacePath := filepath.Join(workspacesDir, workspace)
	fmt.Println(workspacePath)
	if _, err := os.Stat(workspacePath); os.IsNotExist(err) {
		var availableWorkspaces []string

		// List available workspaces
		files, err := os.ReadDir(workspacesDir)
		if err != nil {
			return fmt.Errorf("failed to read workspaces directory: %v", err)
		}
		for _, file := range files {
			if file.IsDir() {
				availableWorkspaces = append(availableWorkspaces, file.Name())
			}
		}

		return fmt.Errorf("invalid workspace: '%s'.\n\nAvailable workspaces are:\n%s", workspace, strings.Join(availableWorkspaces, ", "))
	}
	return nil
}
