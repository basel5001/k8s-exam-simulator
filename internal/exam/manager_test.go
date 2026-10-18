package exam

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func setupTestManager(t *testing.T) *Manager {
	t.Helper()
	stateDir := t.TempDir()
	m, err := NewManager(stateDir)
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}
	// Point scenarios to the real scenarios dir
	wd, _ := os.Getwd()
	m.scenariosDir = filepath.Join(wd, "..", "..", "scenarios")
	return m
}

func TestListScenarios(t *testing.T) {
	m := setupTestManager(t)
	summaries, err := m.ListScenarios()
	if err != nil {
		t.Fatalf("ListScenarios: %v", err)
	}
	if len(summaries) < 15 {
		t.Errorf("expected at least 15 scenarios, got %d", len(summaries))
	}

	// Check all exams are represented
	exams := map[string]int{}
	for _, s := range summaries {
		exams[s.Exam]++
	}
	for _, exam := range []string{"cks", "cka", "ckad"} {
		if exams[exam] < 5 {
			t.Errorf("expected at least 5 %s scenarios, got %d", exam, exams[exam])
		}
	}
}

func TestFindScenarioByName(t *testing.T) {
	m := setupTestManager(t)

	s, err := m.findScenarioByName("network-policy-deny-all")
	if err != nil {
		t.Fatalf("findScenarioByName: %v", err)
	}
	if s.Exam != "cks" {
		t.Errorf("expected exam cks, got %s", s.Exam)
	}
	if s.Name != "network-policy-deny-all" {
		t.Errorf("expected name network-policy-deny-all, got %s", s.Name)
	}
	if s.Validation == "" {
		t.Error("expected non-empty validation script")
	}
	if len(s.Hints) == 0 {
		t.Error("expected at least one hint")
	}
}

func TestFindScenarioNotFound(t *testing.T) {
	m := setupTestManager(t)
	_, err := m.findScenarioByName("nonexistent-scenario")
	if err == nil {
		t.Error("expected error for nonexistent scenario")
	}
}

func TestSessionSaveLoad(t *testing.T) {
	m := setupTestManager(t)

	session := &Session{
		ExamType:      "cks",
		ClusterName:   "kind-cks-exam",
		ScenarioNames: []string{"network-policy-deny-all", "rbac-least-privilege"},
		CurrentIndex:  1,
		StartedAt:     time.Now(),
		Duration:      2 * time.Hour,
		NoTimer:       true,
		Score:         7,
		Results:       map[string]bool{"network-policy-deny-all": true},
		HintIndexes:   map[string]int{"network-policy-deny-all": 2},
	}

	if err := m.saveSession(session); err != nil {
		t.Fatalf("saveSession: %v", err)
	}

	loaded, err := m.LoadSession()
	if err != nil {
		t.Fatalf("LoadSession: %v", err)
	}

	if loaded.ExamType != session.ExamType {
		t.Errorf("ExamType: got %s, want %s", loaded.ExamType, session.ExamType)
	}
	if loaded.NoTimer != true {
		t.Error("NoTimer should be true")
	}
	if loaded.Score != 7 {
		t.Errorf("Score: got %d, want 7", loaded.Score)
	}
	if loaded.CurrentIndex != 1 {
		t.Errorf("CurrentIndex: got %d, want 1", loaded.CurrentIndex)
	}
	if !loaded.Results["network-policy-deny-all"] {
		t.Error("expected network-policy-deny-all to be passed")
	}
	if loaded.HintIndexes["network-policy-deny-all"] != 2 {
		t.Error("expected hint index 2 for network-policy-deny-all")
	}
}

func TestLoadSessionNoFile(t *testing.T) {
	m := setupTestManager(t)
	_, err := m.LoadSession()
	if err == nil {
		t.Error("expected error when no session file exists")
	}
}

func TestStartExamInvalidType(t *testing.T) {
	m := setupTestManager(t)
	_, err := m.StartExam(context.Background(), "invalid", false)
	if err == nil {
		t.Error("expected error for invalid exam type")
	}
}

func TestScenarioYAMLParsing(t *testing.T) {
	m := setupTestManager(t)
	scenarios, err := m.loadAllScenarios()
	if err != nil {
		t.Fatalf("loadAllScenarios: %v", err)
	}

	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			if s.Name == "" {
				t.Error("empty name")
			}
			if s.Exam == "" {
				t.Error("empty exam")
			}
			if s.Weight <= 0 {
				t.Errorf("invalid weight: %d", s.Weight)
			}
			if s.Duration <= 0 {
				t.Errorf("invalid duration: %d", s.Duration)
			}
			if s.Description == "" {
				t.Error("empty description")
			}
			if s.Validation == "" {
				t.Error("empty validation")
			}
			if len(s.Hints) == 0 {
				t.Error("no hints provided")
			}
			if s.Setup == "" {
				t.Error("empty setup script")
			}
			// Validate difficulty is known
			switch s.Difficulty {
			case "easy", "medium", "hard":
			default:
				t.Errorf("unknown difficulty: %s", s.Difficulty)
			}
		})
	}
}

func TestScenarioWeightsPerExam(t *testing.T) {
	m := setupTestManager(t)
	scenarios, err := m.loadAllScenarios()
	if err != nil {
		t.Fatalf("loadAllScenarios: %v", err)
	}

	weights := map[string]int{}
	for _, s := range scenarios {
		weights[s.Exam] += s.Weight
	}

	for exam, total := range weights {
		// Weights should sum to a reasonable percentage (not necessarily 100 with 5 scenarios)
		if total < 20 || total > 100 {
			t.Errorf("exam %s total weight %d seems off", exam, total)
		}
	}
}

func TestScorePercent(t *testing.T) {
	tests := []struct {
		name        string
		score       int
		totalWeight int
		want        int
	}{
		{"perfect cks run reports 100 not 34", 34, 34, 100},
		{"partial run", 14, 34, 41},
		{"nothing passed", 0, 34, 0},
		{"single scenario practice", 7, 7, 100},
		{"legacy session without total falls back to raw", 14, 0, 14},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scorePercent(&Session{Score: tc.score, TotalWeight: tc.totalWeight})
			if got != tc.want {
				t.Errorf("scorePercent(score=%d, total=%d) = %d, want %d", tc.score, tc.totalWeight, got, tc.want)
			}
		})
	}
}

func TestTotalWeightMatchesScenarios(t *testing.T) {
	m := setupTestManager(t)
	for _, examType := range []string{"cks", "cka", "ckad"} {
		scenarios, err := m.loadExamScenarios(examType)
		if err != nil {
			t.Fatalf("loadExamScenarios(%s): %v", examType, err)
		}
		total := totalWeight(scenarios)
		if total <= 0 {
			t.Errorf("exam %s has no total weight", examType)
		}
		// A run passing every scenario must report exactly 100%.
		if got := scorePercent(&Session{Score: total, TotalWeight: total}); got != 100 {
			t.Errorf("exam %s perfect run reports %d%%, want 100%%", examType, got)
		}
	}
}

func TestNoHardcodedClusterNames(t *testing.T) {
	m := setupTestManager(t)
	scenarios, err := m.loadAllScenarios()
	if err != nil {
		t.Fatalf("loadAllScenarios: %v", err)
	}

	// Node names must come from KIND_CLUSTER_NAME so the cluster can be renamed
	// in one place. The context block is display text and is exempt.
	pattern := regexp.MustCompile(`kind-ck[a-z]*-exam-(control-plane|worker)`)
	for _, s := range scenarios {
		t.Run(s.Name, func(t *testing.T) {
			for field, script := range map[string]string{"setup": s.Setup, "validation": s.Validation, "solution": s.Solution} {
				if match := pattern.FindString(script); match != "" {
					t.Errorf("%s hardcodes %q; use ${KIND_CLUSTER_NAME}-control-plane instead", field, match)
				}
			}
		})
	}
}

func TestSessionYAMLRoundTrip(t *testing.T) {
	session := &Session{
		ExamType:      "ckad",
		ClusterName:   "kind-ckad-practice",
		ScenarioNames: []string{"probes", "rolling-update"},
		CurrentIndex:  0,
		StartedAt:     time.Now().Truncate(time.Second),
		Duration:      2 * time.Hour,
		NoTimer:       false,
		PracticeMode:  true,
		Score:         0,
		Results:       map[string]bool{},
		HintIndexes:   map[string]int{},
	}

	data, err := yaml.Marshal(session)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var loaded Session
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if loaded.ExamType != session.ExamType {
		t.Errorf("ExamType mismatch")
	}
	if loaded.PracticeMode != session.PracticeMode {
		t.Error("PracticeMode mismatch")
	}
	if loaded.NoTimer != session.NoTimer {
		t.Error("NoTimer mismatch")
	}
}
