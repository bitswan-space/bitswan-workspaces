package automation

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/bitswan-space/bitswan-workspaces/internal/config"
)

func newRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove the automation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName, err := config.GetWorkspaceName()
			automationDeploymentId := args[0]
			if err != nil {
				return fmt.Errorf("failed to get active workspace from config.toml: %v", err)
			}

			fmt.Printf("Removing an automation %s...\n", automationDeploymentId)
			err = RemoveAutomation(workspaceName, automationDeploymentId)
			if err != nil {
				return fmt.Errorf("failed to remove an automation: %v", err)
			}
			return nil
		},
	}

	return cmd
}

func RemoveAutomation(workspaceName, automationDeploymentId string) error {
	metadata := getMetadata(workspaceName)
	// Construct the URL for stopping the automation
	url := fmt.Sprintf("%s/automations/%s", metadata.GitOpsURL, automationDeploymentId)

	// Send the request to stop the automation
	resp, err := SendAutomationRequest("DELETE", url, metadata.GitOpsSecret)
	if err != nil {
		return fmt.Errorf("failed to send request to remove automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to remove automation, status code: %d", resp.StatusCode)
	}
	fmt.Printf("Automation %s removed successfully.\n", automationDeploymentId)
	return nil
}
