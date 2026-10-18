package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newScenarioCmd() *cobra.Command {
	var noTimer bool

	cmd := &cobra.Command{
		Use:   "scenario [name]",
		Short: "Run a single scenario in practice mode",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			session, err := manager.StartScenario(cmd.Context(), args[0], noTimer)
			if err != nil {
				return err
			}
			fmt.Println(manager.RenderCurrentTask(session))
			return nil
		},
	}

	cmd.Flags().BoolVar(&noTimer, "no-timer", false, "Disable the timer (practice at your own pace)")

	return cmd
}
