package automation

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/bitswan-space/bitswan-workspaces/internal/config"
)

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the automation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName, err := config.GetWorkspaceName()
			automationDeploymentId := args[0]
			if err != nil {
				return fmt.Errorf("failed to get active workspace from config.toml: %v", err)
			}

			fmt.Printf("Starting an automation %s...\n", automationDeploymentId)
			err = startAutomation(workspaceName, automationDeploymentId)
			if err != nil {
				return fmt.Errorf("failed to start an automation: %v", err)
			}
			return nil
		},
	}

	return cmd
}

func startAutomation(workspaceName, automationDeploymentId string) error {
	metadata := getMetadata(workspaceName)
	// Construct the URL for stopping the automation
	url := fmt.Sprintf("%s/automations/%s/start", metadata.GitOpsURL, automationDeploymentId)

	// Send the request to stop the automation
	resp, err := SendAutomationRequest("POST", url, metadata.GitOpsSecret)
	if err != nil {
		return fmt.Errorf("failed to send request to start automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to start automation, status code: %d", resp.StatusCode)
	}
	fmt.Printf("Automation %s started successfully.\n", automationDeploymentId)
	return nil
}
