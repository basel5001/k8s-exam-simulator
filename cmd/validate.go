package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate current task",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			result, session, err := manager.ValidateCurrent(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Println(manager.RenderValidation(result, session))
			return nil
		},
	}
}
