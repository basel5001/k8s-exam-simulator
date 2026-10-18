package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newSkipCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "skip",
		Short: "Skip to next task",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			session, skipped, err := manager.SkipCurrent(cmd.Context())
			if err != nil {
				return err
			}
			fmt.Printf("Skipped: %s\n", skipped)
			fmt.Println(manager.RenderCurrentTask(session))
			return nil
		},
	}
}
