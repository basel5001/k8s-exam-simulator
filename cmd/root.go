package cmd

import (
	"os"
	"path/filepath"

	"github.com/basel5001/k8s-exam-simulator/internal/exam"
	"github.com/spf13/cobra"
)

const appName = "k8s-exam"

var rootCmd = &cobra.Command{
	Use:           appName,
	Short:         "Kubernetes exam simulator",
	Long:          "A practice CLI for running original Kubernetes certification-style scenarios on local kind clusters.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	rootCmd.AddCommand(newStartCmd())
	rootCmd.AddCommand(newListCmd())
	rootCmd.AddCommand(newValidateCmd())
	rootCmd.AddCommand(newHintCmd())
	rootCmd.AddCommand(newSkipCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newTaskCmd())
	rootCmd.AddCommand(newResetCmd())
	rootCmd.AddCommand(newScenarioCmd())
}

func mustManager() *exam.Manager {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	manager, err := exam.NewManager(filepath.Join(cwd, ".k8s-exam-state"))
	if err != nil {
		panic(err)
	}
	return manager
}
