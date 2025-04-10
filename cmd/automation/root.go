package automation

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"
)

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
