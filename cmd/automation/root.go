package automation

import (
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
	cmd.AddCommand(newStopCmd())
	cmd.AddCommand(newStartCmd())
	cmd.AddCommand(newRestartCmd())
	cmd.AddCommand(newRemoveCmd())

	return cmd
}
