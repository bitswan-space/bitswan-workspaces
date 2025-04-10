package automation

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

func newRestartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart the automation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			workspaceName, err := getWorkspaceName()
			automationDeploymentId := args[0]
			if err != nil {
				return fmt.Errorf("failed to get active workspace from config.toml: %v", err)
			}

			fmt.Printf("Restarting an automation %s...\n", automationDeploymentId)
			err = restartAutomation(workspaceName, automationDeploymentId)
			if err != nil {
				return fmt.Errorf("failed to restart an automation: %v", err)
			}
			return nil
		},
	}

	return cmd
}

func restartAutomation(workspaceName, automationDeploymentId string) error {
	metadata := getMetadata(workspaceName)
	// Construct the URL for stopping the automation
	url := fmt.Sprintf("%s/automations/%s/restart", metadata.GitOpsURL, automationDeploymentId)

	// Send the request to stop the automation
	resp, err := SendAutomationRequest("POST", url, metadata.GitOpsSecret)
	if err != nil {
		return fmt.Errorf("failed to send request to restart automation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to restart automation, status code: %d", resp.StatusCode)
	}
	fmt.Printf("Automation %s restarted successfully.\n", automationDeploymentId)
	return nil
}
