package automation

import (
	"fmt"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

type Metadata struct {
	Domain       string `yaml:"domain"`
	EditorURL    string `yaml:"editor-url"`
	GitOpsURL    string `yaml:"gitops-url"`
	GitOpsSecret string `yaml:"gitops-secret"`
}

func NewAutomationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "automation",
		Short: "Manage automations of active workspace",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newRestartCmd())
	cmd.AddCommand(newRemoveCmd())

	return cmd
}

func SendAutomationRequest(method, url string, workspaceSecret string) (*http.Response, error) {
	// Create a new GET request
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Authorization", "Bearer "+workspaceSecret)

	// Create HTTP client and send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	return resp, nil
}

func getMetadata(workspaceName string) Metadata {
	metadataPath := os.Getenv("HOME") + "/.config/bitswan/" + "workspaces/" + workspaceName + "/metadata.yaml"

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		panic(err)
	}

	var metadata Metadata
	err = yaml.Unmarshal(data, &metadata)
	if err != nil {
		panic(err)
	}

	return metadata
}
