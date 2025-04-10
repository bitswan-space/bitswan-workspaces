package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newSelectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "select <workspace>",
		Short: "Select a workspace for activation",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			workspace := args[0]
			bitswanDir := filepath.Join(os.Getenv("HOME"), ".config", "bitswan")

			checkValidWorkspace(workspace, bitswanDir)

			// Proceed to write the active workspace to the config file
			configPath := filepath.Join(bitswanDir, "config.toml")
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
					fmt.Printf("Failed to create config directory: %v\n", err)
					return
				}
				file, err := os.Create(configPath)
				if err != nil {
					fmt.Printf("Failed to create config file: %v\n", err)
					return
				}
				defer file.Close()
				fmt.Println("Config file created at:", configPath)
			} else if err != nil {
				fmt.Printf("Error checking config file: %v\n", err)
				return
			} else {
				fmt.Println("Config file already exists at:", configPath)
				fmt.Println("Overwriting the active workspace...")
			}

			// Write the active workspace to the config file
			configContent := fmt.Sprintf("active_workspace = \"%s\"\n", workspace)
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				fmt.Printf("Failed to write to config file: %v\n", err)
				return
			}

			fmt.Printf("Active workspace set to '%s' in config file.\n", workspace)
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

		return fmt.Errorf("invalid workspace: '%s'. Available workspaces are: %v", workspace, availableWorkspaces)
	}
	return nil
}
