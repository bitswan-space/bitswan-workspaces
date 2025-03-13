package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"

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
			err := updateGitopsDockerImage(gitopsName)
			if err != nil {
				fmt.Printf("Error updating Docker image: %v\n", err)
				return
			}
			fmt.Println("Docker image updated successfully!")
		},
	}
}

// update the docker-compose.yml
func updateDockerCompose(gitopsName string, gitopsImage string, bitswanEditorImage string) error {

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Failed to get home directory:", err)
	}

	dockerComposePath := filepath.Join(homeDir, ".config", "bitswan", gitopsName, "deployment", "docker-compose.yml")

	// Step 1: Read the file
	dockerComposeData, err := os.ReadFile(dockerComposePath)
	if err != nil {
		return fmt.Errorf("Failed to read docker-compose.yml:", err)
	}

	// Step 2: Parse the YAML
	var composeConfig map[string]interface{}
	err = yaml.Unmarshal(dockerComposeData, &composeConfig)
	if err != nil {
		return fmt.Errorf("Error parsing YAML:", err)
	}

	// Step 3: Update the image of gitops service
	if service, ok := composeConfig["services"].(map[string]interface{}); ok {
		if serviceMap, ok := service[gitopsImage].(map[string]interface{}); ok {
			if gitopsImage, ok := serviceMap["image"].(string); ok {
				serviceMap["image"] = gitopsImage
			}
		}
	}

	// Step 4: Update the image of editor service
	if service, ok := composeConfig["services"].(map[string]interface{}); ok {
		if serviceMap, ok := service["bitswan-editor-"+gitopsName].(map[string]interface{}); ok {
			if bitswanEditorImage, ok := serviceMap["image"].(string); ok {
				serviceMap["image"] = bitswanEditorImage
			}
		}
	}

	// Step 4: Convert back to YAML
	updatedData, err := yaml.Marshal(&composeConfig)
	if err != nil {
		return fmt.Errorf("Error marshalling YAML:", err)
	}

	// Step 5: Write back to file
	err = os.WriteFile(dockerComposePath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("Error writing file:", err)
	}

	return nil
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

func updateGitopsDockerImage(gitopsName string) error {
	gitopsImage, bitswanEditorImage := getLatestImagesVersion()

	// Update the docker-compose.yml
	err := updateDockerCompose(gitopsName, gitopsImage, bitswanEditorImage)
	if err != nil {
		panic(fmt.Errorf("failed to update Docker image"))
	}

	fmt.Println("Updated docker-compose.yaml successfully!")

	dir := fmt.Sprintf("~/.config/bitswan/%s/deployment", gitopsName)
	cmd := exec.Command("sh", "-c", fmt.Sprintf("cd %s && docker-compose pull", dir))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update Docker image: %v, output: %s", err, string(output))
	}

	cmd = exec.Command("sh", "-c", fmt.Sprintf("cd %s && docker-compose up -d", dir))
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to update Docker image: %v, output: %s", err, string(output))
	}

	return nil
}
