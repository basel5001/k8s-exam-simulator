package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show exam progress and score",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			session, err := manager.LoadSession()
			if err != nil {
				return err
			}
			fmt.Println(manager.RenderStatus(session))
			return nil
		},
	}
}
