package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newTaskCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "task",
		Short:   "Re-display the current task description",
		Aliases: []string{"question", "show"},
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			session, err := manager.LoadSession()
			if err != nil {
				return err
			}
			fmt.Println(manager.RenderCurrentTask(session))
			return nil
		},
	}
}
