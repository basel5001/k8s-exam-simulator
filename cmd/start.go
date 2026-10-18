package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var noTimer bool

	cmd := &cobra.Command{
		Use:   "start [cks|cka|ckad]",
		Short: "Start an exam session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			session, err := manager.StartExam(cmd.Context(), args[0], noTimer)
			if err != nil {
				return err
			}
			fmt.Println(manager.RenderCurrentTask(session))
			return nil
		},
	}

	cmd.Flags().BoolVar(&noTimer, "no-timer", false, "Disable the exam timer (practice at your own pace)")

	return cmd
}
