package validator

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Run executes a validation script with KIND_CLUSTER_NAME exported, so scenario
// scripts can address nodes as "${KIND_CLUSTER_NAME}-control-plane" instead of
// hardcoding a cluster name.
func Run(ctx context.Context, clusterName, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "bash", "-c", strings.TrimSpace(script))
	cmd.Env = ScriptEnv(clusterName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	output := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	if err != nil {
		return output, fmt.Errorf("validation failed: %w", err)
	}
	return output, nil
}

// ScriptEnv builds the environment shared by setup and validation scripts.
func ScriptEnv(clusterName string) []string {
	return append(os.Environ(),
		"KIND_CLUSTER_NAME="+clusterName,
		"KUBECONFIG="+filepath.Join(os.Getenv("HOME"), ".kube", "config"),
	)
}
