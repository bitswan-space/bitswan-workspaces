package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/bitswan-space/bitswan-gitops-cli/internal/dockercompose"
	"github.com/bitswan-space/bitswan-gitops-cli/internal/dockerhub"
	"github.com/spf13/cobra"
)

type updateOptions struct {
	gitopsImage string
	editorImage string
}

func newUpdateCmd() *cobra.Command {
	o := &updateOptions{}
	cmd := &cobra.Command{
		Use:          "update <gitops-name>",
		Short:        "bitswan-gitops update",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			gitopsName := args[0]
			fmt.Printf("Updating Gitops: %s...\n", gitopsName)
			err := updateGitops(gitopsName, o)
			if err != nil {
				fmt.Errorf("Error updating gitops: %w", err)
				return
			}
			fmt.Printf("Gitops %s updated successfully!\n", gitopsName)
		},
	}

	cmd.Flags().StringVar(&o.gitopsImage, "gitops-image", "", "Custom image for the gitops")
	cmd.Flags().StringVar(&o.editorImage, "editor-image", "", "Custom image for the editor")

	return cmd
}

func updateGitops(gitopsName string, o *updateOptions) error {
	bitswanPath := os.Getenv("HOME") + "/.config/bitswan/"

	repoPath := filepath.Join(bitswanPath, "bitswan-src")
	// 1. Create or update examples directory
	fmt.Println("Ensuring examples are up to date...")
	err := EnsureExamples(repoPath, true)
	if err != nil {
		return fmt.Errorf("Failed to download examples: %w", err)
	}
	fmt.Println("Examples are up to date!")

	// 2. Update Docker images and docker-compose file
	fmt.Println("Updating Docker images and docker-compose file...")
	gitopsImage := o.gitopsImage
	if gitopsImage == "" {
		gitopsLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/gitops/tags/")
		if err != nil {
			panic(fmt.Errorf("Failed to get latest BitSwan GitOps version: %w", err))
		}
		gitopsImage = "bitswan/gitops:" + gitopsLatestVersion
	}

	bitswanEditorImage := o.editorImage
	if o.editorImage == "" {
		bitswanEditorLatestVersion, err := dockerhub.GetLatestDockerHubVersion("https://hub.docker.com/v2/repositories/bitswan/bitswan-editor/tags/")
		if err != nil {
			panic(fmt.Errorf("failed to get latest BitSwan Editor version: %w", err))
		}
		bitswanEditorImage = "bitswan/bitswan-editor:" + bitswanEditorLatestVersion
	}

	gitopsConfig := filepath.Join(bitswanPath, "workspaces/", gitopsName)

	// Get the domain from the file `~/.config/bitswan/<gitops-name>/deployment/domain`
	dataPath := filepath.Join(os.Getenv("HOME"), ".config", "bitswan", "workspaces", gitopsName, "metadata.yaml")

	data, err := os.ReadFile(dataPath)
	if err != nil {
		return fmt.Errorf("Failed to read metadata.yaml: %w", err)
	}

	// Config represents the structure of the YAML file
	var metadata struct {
		Domain       string `yaml:"domain"`
		EditorURL    string `yaml:"editor-url"`
		GitopsURL    string `yaml:"gitops-url"`
		GitopsSecret string `yaml:"gitops-secret"`
	}

	if err := yaml.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("Failed to unmarshal metadata.yaml: %w", err)
	}

	// Rewrite the docker-compose file
	noIde := metadata.EditorURL == ""
	compose, _, err := dockercompose.CreateDockerComposeFile(gitopsConfig, gitopsName, gitopsImage, bitswanEditorImage, metadata.Domain, noIde)
	if err != nil {
		panic(fmt.Errorf("Failed to create docker-compose file: %w", err))
	}

	dockerComposeFilePath := filepath.Join(gitopsConfig, "deployment", "/docker-compose.yml")
	if err := os.WriteFile(dockerComposeFilePath, []byte(compose), 0755); err != nil {
		panic(fmt.Errorf("Failed to write docker-compose file: %w", err))
	}

	// 3. Restart gitops and editor services
	fmt.Println("Restarting services...")
	dockerComposePath := filepath.Join(gitopsConfig, "deployment", "docker-compose")
	err = exec.Command(dockerComposePath, "up").Run()
	if err != nil {
		return fmt.Errorf("Failed to stop the services: %w", err)
	}
	fmt.Println("Services restarted!")

	return nil
}
