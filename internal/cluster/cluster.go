package cluster

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/basel5001/k8s-exam-simulator/internal/validator"
)

func CheckPrerequisites(ctx context.Context) error {
	for _, binary := range []string{"kind", "kubectl", "docker"} {
		if _, err := exec.LookPath(binary); err != nil {
			return fmt.Errorf("missing prerequisite %s in PATH", binary)
		}
	}
	return nil
}

func Create(ctx context.Context, examType, clusterName string) error {
	configPath, err := ensureConfig(examType)
	if err != nil {
		return err
	}
	exists, err := Exists(ctx, clusterName)
	if err != nil {
		return err
	}
	if exists {
		if err := reuse(ctx, clusterName); err == nil {
			return nil
		}
		if err := Delete(ctx, clusterName); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, "kind", "create", "cluster", "--name", clusterName, "--config", configPath, "--wait", "120s")
	cmd.Env = validator.ScriptEnv(clusterName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if alreadyExists(string(output)) {
			return reuse(ctx, clusterName)
		}
		return fmt.Errorf("create cluster: %w\n%s", err, output)
	}
	return nil
}

func Exists(ctx context.Context, clusterName string) (bool, error) {
	cmd := exec.CommandContext(ctx, "kind", "get", "clusters")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("list clusters: %w\n%s", err, output)
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == clusterName {
			return true, nil
		}
	}
	return false, nil
}

// reuse adopts an existing kind cluster, refreshing the kubeconfig entry and
// confirming the API server answers before the caller treats it as usable.
func reuse(ctx context.Context, clusterName string) error {
	export := exec.CommandContext(ctx, "kind", "export", "kubeconfig", "--name", clusterName)
	export.Env = validator.ScriptEnv(clusterName)
	if output, err := export.CombinedOutput(); err != nil {
		return fmt.Errorf("export kubeconfig for %s: %w\n%s", clusterName, err, output)
	}
	info := exec.CommandContext(ctx, "kubectl", "cluster-info", "--context", "kind-"+clusterName, "--request-timeout=15s")
	info.Env = validator.ScriptEnv(clusterName)
	if output, err := info.CombinedOutput(); err != nil {
		return fmt.Errorf("cluster %s unreachable: %w\n%s", clusterName, err, output)
	}
	return nil
}

func alreadyExists(output string) bool {
	return strings.Contains(output, "already exists") || strings.Contains(output, "already exist for a cluster")
}

func Delete(ctx context.Context, clusterName string) error {
	cmd := exec.CommandContext(ctx, "kind", "delete", "cluster", "--name", clusterName)
	cmd.Env = validator.ScriptEnv(clusterName)
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "not found") {
		return fmt.Errorf("delete cluster: %w\n%s", err, output)
	}
	return nil
}

func RunScript(ctx context.Context, clusterName, script string) error {
	trimmed := strings.TrimSpace(script)
	if trimmed == "" {
		return nil
	}
	cmd := exec.CommandContext(ctx, "bash", "-c", trimmed)
	cmd.Env = validator.ScriptEnv(clusterName)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("script failed: %w\n%s%s", err, stdout.String(), stderr.String())
	}
	return nil
}

func ensureConfig(examType string) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	path := filepath.Join(wd, "internal", "cluster", fmt.Sprintf("%s-kind.yaml", examType))
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("cluster config for %s not found: %w", examType, err)
	}
	return path, nil
}
