package cmd

import "github.com/spf13/cobra"

func newResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Tear down cluster and reset session",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			return manager.Reset(cmd.Context())
		},
	}
}
