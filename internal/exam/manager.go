package exam

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/basel5001/k8s-exam-simulator/internal/cluster"
	"github.com/basel5001/k8s-exam-simulator/internal/ui"
	"github.com/basel5001/k8s-exam-simulator/internal/validator"
	"gopkg.in/yaml.v3"
)

var examDurations = map[string]time.Duration{
	"cks":  2 * time.Hour,
	"cka":  2 * time.Hour,
	"ckad": 2 * time.Hour,
}

type Manager struct {
	stateDir     string
	sessionFile  string
	scenariosDir string
	renderer     *ui.Renderer
}

func NewManager(stateDir string) (*Manager, error) {
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("resolve working directory: %w", err)
	}
	return &Manager{
		stateDir:     stateDir,
		sessionFile:  filepath.Join(stateDir, "session.yaml"),
		scenariosDir: filepath.Join(wd, "scenarios"),
		renderer:     ui.NewRenderer(),
	}, nil
}

func (m *Manager) ListScenarios() ([]ScenarioSummary, error) {
	scenarios, err := m.loadAllScenarios()
	if err != nil {
		return nil, err
	}
	summaries := make([]ScenarioSummary, 0, len(scenarios))
	for _, scenario := range scenarios {
		summaries = append(summaries, ScenarioSummary{Name: scenario.Name, Exam: scenario.Exam, Weight: scenario.Weight, Difficulty: scenario.Difficulty, Duration: scenario.Duration, Domains: scenario.Domains, Description: scenario.Description})
	}
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Exam == summaries[j].Exam {
			return summaries[i].Name < summaries[j].Name
		}
		return summaries[i].Exam < summaries[j].Exam
	})
	return summaries, nil
}

func (m *Manager) StartExam(ctx context.Context, examType string, noTimer bool) (*Session, error) {
	if _, ok := examDurations[examType]; !ok {
		return nil, fmt.Errorf("unsupported exam type %q", examType)
	}
	scenarios, err := m.loadExamScenarios(examType)
	if err != nil {
		return nil, err
	}
	if len(scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios found for exam %s", examType)
	}
	clusterName := fmt.Sprintf("kind-%s-exam", examType)
	if err := cluster.CheckPrerequisites(ctx); err != nil {
		return nil, err
	}
	if err := cluster.Create(ctx, examType, clusterName); err != nil {
		return nil, err
	}
	session := &Session{
		ExamType:      examType,
		ClusterName:   clusterName,
		ScenarioNames: scenarioNames(scenarios),
		CurrentIndex:  0,
		StartedAt:     time.Now(),
		Duration:      examDurations[examType],
		NoTimer:       noTimer,
		TotalWeight:   totalWeight(scenarios),
		Results:       map[string]bool{},
		HintIndexes:   map[string]int{},
	}
	if err := m.saveSession(session); err != nil {
		return nil, err
	}
	if err := m.prepareCurrentScenario(ctx, session); err != nil {
		return nil, err
	}
	return session, nil
}

func (m *Manager) StartScenario(ctx context.Context, name string, noTimer bool) (*Session, error) {
	scenario, err := m.findScenarioByName(name)
	if err != nil {
		return nil, err
	}
	// Practice sessions reuse the exam cluster name because scenario scripts
	// address nodes as kind-<exam>-exam-control-plane, and only one session
	// exists at a time.
	clusterName := fmt.Sprintf("kind-%s-exam", scenario.Exam)
	if err := cluster.CheckPrerequisites(ctx); err != nil {
		return nil, err
	}
	if err := cluster.Create(ctx, scenario.Exam, clusterName); err != nil {
		return nil, err
	}
	session := &Session{
		ExamType:      scenario.Exam,
		ClusterName:   clusterName,
		ScenarioNames: []string{scenario.Name},
		CurrentIndex:  0,
		StartedAt:     time.Now(),
		Duration:      time.Duration(scenario.Duration) * time.Minute,
		NoTimer:       noTimer,
		PracticeMode:  true,
		TotalWeight:   scenario.Weight,
		Results:       map[string]bool{},
		HintIndexes:   map[string]int{},
	}
	if err := m.saveSession(session); err != nil {
		return nil, err
	}
	if err := cluster.RunScript(ctx, session.ClusterName, scenario.Setup); err != nil {
		return nil, fmt.Errorf("setup scenario %s: %w", scenario.Name, err)
	}
	return session, nil
}

func (m *Manager) ValidateCurrent(ctx context.Context) (*ValidationResult, *Session, error) {
	session, err := m.LoadSession()
	if err != nil {
		return nil, nil, err
	}
	scenario, err := m.currentScenario(session)
	if err != nil {
		return nil, nil, err
	}
	output, err := validator.Run(ctx, session.ClusterName, scenario.Validation)
	passed := err == nil
	if passed && !session.Results[scenario.Name] {
		session.Results[scenario.Name] = true
		session.Score += scenario.Weight
		if session.CurrentIndex < len(session.ScenarioNames)-1 {
			session.CurrentIndex++
			if err := m.prepareCurrentScenario(ctx, session); err != nil {
				return nil, nil, err
			}
		} else {
			now := time.Now()
			session.CompletedAt = &now
		}
		if err := m.saveSession(session); err != nil {
			return nil, nil, err
		}
	}
	if !passed {
		session.Results[scenario.Name] = false
		if err := m.saveSession(session); err != nil {
			return nil, nil, err
		}
	}
	return &ValidationResult{Passed: passed, Output: output, Scenario: scenario}, session, nil
}

func (m *Manager) CurrentHint() (string, error) {
	session, err := m.LoadSession()
	if err != nil {
		return "", err
	}
	scenario, err := m.currentScenario(session)
	if err != nil {
		return "", err
	}
	idx := session.HintIndexes[scenario.Name]
	if idx >= len(scenario.Hints) {
		return "No more hints available.", nil
	}
	hint := fmt.Sprintf("Hint %d/%d: %s", idx+1, len(scenario.Hints), scenario.Hints[idx])
	session.HintIndexes[scenario.Name] = idx + 1
	if err := m.saveSession(session); err != nil {
		return "", err
	}
	return hint, nil
}

func (m *Manager) SkipCurrent(ctx context.Context) (*Session, string, error) {
	session, err := m.LoadSession()
	if err != nil {
		return nil, "", err
	}
	scenario, err := m.currentScenario(session)
	if err != nil {
		return nil, "", err
	}
	skipped := scenario.Name
	if session.CurrentIndex >= len(session.ScenarioNames)-1 {
		now := time.Now()
		session.CompletedAt = &now
	} else {
		session.CurrentIndex++
		if err := m.prepareCurrentScenario(ctx, session); err != nil {
			return nil, "", err
		}
	}
	if err := m.saveSession(session); err != nil {
		return nil, "", err
	}
	return session, skipped, nil
}

func (m *Manager) LoadSession() (*Session, error) {
	data, err := os.ReadFile(m.sessionFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("no active session; run k8s-exam start [cks|cka|ckad] or k8s-exam scenario [name]")
		}
		return nil, fmt.Errorf("read session: %w", err)
	}
	var session Session
	if err := yaml.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("parse session: %w", err)
	}
	if session.Results == nil {
		session.Results = map[string]bool{}
	}
	if session.HintIndexes == nil {
		session.HintIndexes = map[string]int{}
	}
	return &session, nil
}

func (m *Manager) Reset(ctx context.Context) error {
	session, err := m.LoadSession()
	if err == nil {
		if destroyErr := cluster.Delete(ctx, session.ClusterName); destroyErr != nil {
			return destroyErr
		}
	}
	if removeErr := os.Remove(m.sessionFile); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
		return fmt.Errorf("remove session: %w", removeErr)
	}
	fmt.Println("Session reset complete.")
	return nil
}

func (m *Manager) RenderCurrentTask(session *Session) string {
	scenario, err := m.currentScenario(session)
	if err != nil {
		return err.Error()
	}
	return m.renderer.RenderTask(ui.TaskView{
		ExamType:      session.ExamType,
		CurrentIndex:  session.CurrentIndex,
		Total:         len(session.ScenarioNames),
		Score:         scorePercent(session),
		StartedAt:     session.StartedAt,
		Duration:      session.Duration,
		NoTimer:       session.NoTimer,
		ScenarioName:  scenario.Name,
		Domains:       scenario.Domains,
		Difficulty:    scenario.Difficulty,
		EstimateMins:  scenario.Duration,
		Description:   scenario.Description,
		Context:       scenario.Context,
		Completed:     session.CompletedAt != nil,
		ValidatedTask: len(session.Results),
	})
}

func (m *Manager) RenderValidation(result *ValidationResult, session *Session) string {
	return m.renderer.RenderValidation(ui.ValidationView{Passed: result.Passed, ScenarioName: result.Scenario.Name, Output: result.Output, Completed: session.CompletedAt != nil})
}

func (m *Manager) RenderStatus(session *Session) string {
	return m.renderer.RenderStatus(ui.StatusView{ExamType: session.ExamType, CurrentIndex: session.CurrentIndex, Total: len(session.ScenarioNames), Score: scorePercent(session), StartedAt: session.StartedAt, Duration: session.Duration, NoTimer: session.NoTimer, Completed: session.CompletedAt != nil, ValidatedTasks: len(session.Results)})
}

func (m *Manager) currentScenario(session *Session) (Scenario, error) {
	if session.CurrentIndex >= len(session.ScenarioNames) {
		return Scenario{}, errors.New("exam complete")
	}
	return m.findScenarioByName(session.ScenarioNames[session.CurrentIndex])
}

func (m *Manager) prepareCurrentScenario(ctx context.Context, session *Session) error {
	scenario, err := m.currentScenario(session)
	if err != nil {
		return err
	}
	return cluster.RunScript(ctx, session.ClusterName, scenario.Setup)
}

func (m *Manager) saveSession(session *Session) error {
	data, err := yaml.Marshal(session)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	if err := os.WriteFile(m.sessionFile, data, 0o644); err != nil {
		return fmt.Errorf("write session: %w", err)
	}
	return nil
}

func (m *Manager) loadAllScenarios() ([]Scenario, error) {
	files := []string{}
	for _, examType := range []string{"cks", "cka", "ckad"} {
		matches, err := filepath.Glob(filepath.Join(m.scenariosDir, examType, "*.yaml"))
		if err != nil {
			return nil, err
		}
		files = append(files, matches...)
	}
	result := make([]Scenario, 0, len(files))
	for _, file := range files {
		scenario, err := loadScenario(file)
		if err != nil {
			return nil, err
		}
		result = append(result, scenario)
	}
	return result, nil
}

func (m *Manager) loadExamScenarios(examType string) ([]Scenario, error) {
	matches, err := filepath.Glob(filepath.Join(m.scenariosDir, examType, "*.yaml"))
	if err != nil {
		return nil, err
	}
	result := make([]Scenario, 0, len(matches))
	for _, file := range matches {
		scenario, err := loadScenario(file)
		if err != nil {
			return nil, err
		}
		result = append(result, scenario)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (m *Manager) findScenarioByName(name string) (Scenario, error) {
	scenarios, err := m.loadAllScenarios()
	if err != nil {
		return Scenario{}, err
	}
	for _, scenario := range scenarios {
		if scenario.Name == name {
			return scenario, nil
		}
	}
	return Scenario{}, fmt.Errorf("scenario %q not found", name)
}

func loadScenario(path string) (Scenario, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Scenario{}, fmt.Errorf("read scenario %s: %w", path, err)
	}
	var scenario Scenario
	if err := yaml.Unmarshal(data, &scenario); err != nil {
		return Scenario{}, fmt.Errorf("parse scenario %s: %w", path, err)
	}
	if scenario.Name == "" || scenario.Exam == "" || strings.TrimSpace(scenario.Validation) == "" {
		return Scenario{}, fmt.Errorf("scenario %s missing required fields", path)
	}
	return scenario, nil
}

func scenarioNames(scenarios []Scenario) []string {
	result := make([]string, 0, len(scenarios))
	for _, scenario := range scenarios {
		result = append(result, scenario.Name)
	}
	return result
}

func totalWeight(scenarios []Scenario) int {
	total := 0
	for _, scenario := range scenarios {
		total += scenario.Weight
	}
	return total
}

// scorePercent normalises earned weight against the weight available in the
// session. Scenario weights are a share of the full certification blueprint,
// so a five-task exam sums to roughly a third and would otherwise report a
// perfect run as 34%.
func scorePercent(session *Session) int {
	if session.TotalWeight <= 0 {
		return session.Score
	}
	return session.Score * 100 / session.TotalWeight
}
