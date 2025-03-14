package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/bitswan-space/bitswan-gitops-cli/internal/dockercompose"
	"github.com/bitswan-space/bitswan-gitops-cli/internal/dockerhub"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:          "update <gitops-name>",
		Short:        "bitswan-gitops update",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			gitopsName := args[0]
			fmt.Printf("Updating Docker image: %s...\n", gitopsName)
			err := updateGitops(gitopsName)
			if err != nil {
				fmt.Errorf("Error updating gitops: %v\n", err)
				return
			}
			fmt.Println("Docker image updated successfully!")
		},
	}
}

func getLatestImagesVersion() (string, string) {
	gitopsLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/gitops/tags/")
	if err != nil {
		panic(fmt.Errorf("failed to get latest BitSwan GitOps version: %w", err))
	}
	gitopsImage := "bitswan/gitops:" + gitopsLatestVersion

	bitswanEditorLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/bitswan-editor/tags/")
	if err != nil {
		panic(fmt.Errorf("failed to get latest BitSwan Editor version: %w", err))
	}
	bitswanEditorImage := "bitswan/bitswan-editor:" + bitswanEditorLatestVersion

	return gitopsImage, bitswanEditorImage
}

func updateGitops(gitopsName string) error {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"

	repoPath := filepath.Join(bitswanPath, "bitswan-src")
	// 1. Create or update examples directory
	err := EnsureExamples(repoPath, true)
	if err != nil {
		return fmt.Errorf("Failed to download examples: %w", err)
	}

	// 2. Update Docker images and docker-compose file
	gitopsImage, bitswanEditorImage := getLatestImagesVersion()
	gitopsConfig := filepath.Join(bitswanPath, "workspaces/", gitopsName)

	// Get the domain from the file `~/.config/bitswan/<gitops-name>/deployment/domain`
	dataPath := filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "workspaces", gitopsName, "metadata.yaml")

	data, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("Failed to read metadata.yaml:", err)
	}

	// Config represents the structure of the YAML file
	var metadata struct {
		Domain       string `yaml:"domain"`
		EditorURL    string `yaml:"editor-url"`
		GitopsURL    string `yaml:"gitops-url"`
		GitopsSecret string `yaml:"gitops-secret"`
	}

	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("Failed to unmarshal metadata.yaml:", err)
	}

	// Rewrite the docker-compose file
	noIde := metadata.EditorURL == ""
	dockercompose.CreateDockerComposeFile(gitopsConfig, gitopsName, gitopsImage, bitswanEditorImage, metadata.Domain, noIde)

	// 3. Restart gitops and editor services

	return nil
}
