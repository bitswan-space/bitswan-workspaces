package caddy

import "github.com/spf13/cobra"

func NewCaddyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "caddy",
		Short: "Manage caddy",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	cmd.AddCommand(newInitCmd())

	return cmd
}
