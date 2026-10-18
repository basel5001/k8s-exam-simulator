package validator

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRunPassingScript(t *testing.T) {
	output, err := Run(context.Background(), "kind-test-exam", `echo "PASS: check completed"`)
	if err != nil {
		t.Fatalf("expected pass, got error: %v", err)
	}
	if output == "" {
		t.Error("expected non-empty output")
	}
}

func TestRunFailingScript(t *testing.T) {
	output, err := Run(context.Background(), "kind-test-exam", `echo "FAIL: something wrong"; exit 1`)
	if err == nil {
		t.Fatal("expected error for failing script")
	}
	if output == "" {
		t.Error("expected non-empty output even on failure")
	}
}

func TestRunEmptyScript(t *testing.T) {
	output, err := Run(context.Background(), "kind-test-exam", "")
	// Empty script with bash -c "" exits 0
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = output
}

func TestRunTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := Run(ctx, "kind-test-exam", "sleep 10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestRunExportsClusterName(t *testing.T) {
	output, err := Run(context.Background(), "kind-cks-exam", `set -euo pipefail; echo "node=${KIND_CLUSTER_NAME}-control-plane"`)
	if err != nil {
		t.Fatalf("expected pass, got error: %v (output %q)", err, output)
	}
	if !strings.Contains(output, "node=kind-cks-exam-control-plane") {
		t.Errorf("expected expanded node name, got %q", output)
	}
}

func TestRunMultipleChecks(t *testing.T) {
	script := `
echo "Check 1: verifying..."
echo "PASS: check 1"
echo "Check 2: verifying..."
echo "PASS: check 2"
exit 0
`
	output, err := Run(context.Background(), "kind-test-exam", script)
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
	if output == "" {
		t.Error("expected output from multiple checks")
	}
}
