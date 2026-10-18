package ui

import (
	"strings"
	"testing"
	"time"
)

func TestRenderTaskWithTimer(t *testing.T) {
	r := NewRenderer()
	output := r.RenderTask(TaskView{
		ExamType:     "cks",
		CurrentIndex: 0,
		Total:        5,
		Score:        0,
		StartedAt:    time.Now(),
		Duration:     2 * time.Hour,
		NoTimer:      false,
		ScenarioName: "test-scenario",
		Domains:      []string{"Cluster Hardening"},
		Difficulty:   "medium",
		EstimateMins: 10,
		Description:  "Test description",
		Context:      "Cluster: kind-cks-exam",
	})

	if !strings.Contains(output, "CKS exam session") {
		t.Error("expected exam type in output")
	}
	if !strings.Contains(output, "test-scenario") {
		t.Error("expected scenario name in output")
	}
	if strings.Contains(output, "no timer") {
		t.Error("should show timer, not 'no timer'")
	}
}

func TestRenderTaskNoTimer(t *testing.T) {
	r := NewRenderer()
	output := r.RenderTask(TaskView{
		ExamType:     "cka",
		CurrentIndex: 2,
		Total:        5,
		Score:        14,
		StartedAt:    time.Now(),
		Duration:     2 * time.Hour,
		NoTimer:      true,
		ScenarioName: "etcd-backup",
		Domains:      []string{"Storage"},
		Difficulty:   "hard",
		EstimateMins: 12,
		Description:  "Backup etcd",
		Context:      "Cluster: kind-cka-exam",
	})

	if !strings.Contains(output, "no timer") {
		t.Error("expected 'no timer' in output")
	}
	if !strings.Contains(output, "CKA exam session") {
		t.Error("expected CKA in output")
	}
}

func TestRenderValidationPass(t *testing.T) {
	r := NewRenderer()
	output := r.RenderValidation(ValidationView{
		Passed:       true,
		ScenarioName: "test-scenario",
		Output:       "PASS: all checks passed",
		Completed:    false,
	})

	if !strings.Contains(output, "PASS") {
		t.Error("expected PASS in output")
	}
	if !strings.Contains(output, "Next task is ready") {
		t.Error("expected next task message")
	}
}

func TestRenderValidationFail(t *testing.T) {
	r := NewRenderer()
	output := r.RenderValidation(ValidationView{
		Passed:       false,
		ScenarioName: "test-scenario",
		Output:       "FAIL: NetworkPolicy not found",
		Completed:    false,
	})

	if !strings.Contains(output, "FAIL") {
		t.Error("expected FAIL in output")
	}
	if !strings.Contains(output, "Continue working") {
		t.Error("expected continue message")
	}
}

func TestRenderStatusNoTimer(t *testing.T) {
	r := NewRenderer()
	output := r.RenderStatus(StatusView{
		ExamType:     "ckad",
		CurrentIndex: 3,
		Total:        5,
		Score:        20,
		StartedAt:    time.Now(),
		Duration:     2 * time.Hour,
		NoTimer:      true,
		Completed:    false,
	})

	if !strings.Contains(output, "no timer") {
		t.Error("expected 'no timer' in output")
	}
	if !strings.Contains(output, "CKAD") {
		t.Error("expected CKAD in output")
	}
}

func TestRenderProgress(t *testing.T) {
	tests := []struct {
		current   int
		total     int
		completed bool
		wantPct   string
	}{
		{0, 5, false, "0%"},
		{3, 5, false, "60%"},
		{5, 5, true, "100%"},
		{0, 0, false, "0%"},
	}

	for _, tc := range tests {
		result := renderProgress(tc.current, tc.total, tc.completed)
		if !strings.Contains(result, tc.wantPct) {
			t.Errorf("renderProgress(%d, %d, %v) = %q, want containing %q", tc.current, tc.total, tc.completed, result, tc.wantPct)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input time.Duration
		want  string
	}{
		{2 * time.Hour, "02:00:00"},
		{90 * time.Minute, "01:30:00"},
		{0, "00:00:00"},
		{time.Hour + 5*time.Minute + 30*time.Second, "01:05:30"},
	}

	for _, tc := range tests {
		got := formatDuration(tc.input)
		if got != tc.want {
			t.Errorf("formatDuration(%v) = %q, want %q", tc.input, got, tc.want)
		}
	}
}
