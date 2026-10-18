package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newHintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "hint",
		Short: "Show hint for current task",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			hint, err := manager.CurrentHint()
			if err != nil {
				return err
			}
			fmt.Println(hint)
			return nil
		},
	}
}
