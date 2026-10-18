package main

import (
	"fmt"
	"os"

	"github.com/basel5001/k8s-exam-simulator/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
