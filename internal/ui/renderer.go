package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type TaskView struct {
	ExamType      string
	CurrentIndex  int
	Total         int
	Score         int
	StartedAt     time.Time
	Duration      time.Duration
	NoTimer       bool
	ScenarioName  string
	Domains       []string
	Difficulty    string
	EstimateMins  int
	Description   string
	Context       string
	Completed     bool
	ValidatedTask int
}

type ValidationView struct {
	Passed       bool
	ScenarioName string
	Output       string
	Completed    bool
}

type StatusView struct {
	ExamType       string
	CurrentIndex   int
	Total          int
	Score          int
	StartedAt      time.Time
	Duration       time.Duration
	NoTimer        bool
	Completed      bool
	ValidatedTasks int
}

type Renderer struct {
	borderStyle lipgloss.Style
	titleStyle  lipgloss.Style
	passStyle   lipgloss.Style
	failStyle   lipgloss.Style
}

type timerModel struct {
	remaining time.Duration
}

func (m timerModel) Init() tea.Cmd                           { return nil }
func (m timerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m timerModel) View() string                            { return formatDuration(m.remaining) }

func NewRenderer() *Renderer {
	return &Renderer{
		borderStyle: lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1),
		titleStyle:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("63")),
		passStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true),
		failStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
	}
}

func (r *Renderer) RenderTask(view TaskView) string {
	progress := renderProgress(view.CurrentIndex, view.Total, view.Completed)
	timerStr := "∞ (no timer)"
	if !view.NoTimer {
		timerStr = renderTimer(view.StartedAt, view.Duration)
	}
	content := strings.Join([]string{
		r.titleStyle.Render(fmt.Sprintf("%s exam session", strings.ToUpper(view.ExamType))),
		fmt.Sprintf("Task %d/%d • Score %d%% • Time left %s", min(view.CurrentIndex+1, max(1, view.Total)), view.Total, view.Score, timerStr),
		progress,
		"",
		fmt.Sprintf("Scenario: %s", view.ScenarioName),
		fmt.Sprintf("Domains: %s", strings.Join(view.Domains, ", ")),
		fmt.Sprintf("Difficulty: %s • Estimated: %d min", view.Difficulty, view.EstimateMins),
		"",
		"Description:",
		view.Description,
		"",
		"Context:",
		view.Context,
		"",
		"Use kubectl/vim against the kind cluster, then run `k8s-exam validate`.",
	}, "\n")
	return r.borderStyle.Render(content)
}

func (r *Renderer) RenderValidation(view ValidationView) string {
	status := r.failStyle.Render("FAIL")
	if view.Passed {
		status = r.passStyle.Render("PASS")
	}
	next := "Continue working on the current task."
	if view.Passed {
		if view.Completed {
			next = "Exam complete. Review your score with `k8s-exam status`."
		} else {
			next = "Current task completed. Next task is ready."
		}
	}
	return r.borderStyle.Render(strings.Join([]string{
		fmt.Sprintf("Validation result: %s", status),
		fmt.Sprintf("Scenario: %s", view.ScenarioName),
		"",
		view.Output,
		"",
		next,
	}, "\n"))
}

func (r *Renderer) RenderStatus(view StatusView) string {
	remaining := "∞ (no timer)"
	if !view.NoTimer {
		remaining = renderTimer(view.StartedAt, view.Duration)
	}
	status := []string{
		r.titleStyle.Render("Exam status"),
		fmt.Sprintf("Exam: %s", strings.ToUpper(view.ExamType)),
		fmt.Sprintf("Progress: %d/%d", min(view.CurrentIndex+1, max(1, view.Total)), view.Total),
		fmt.Sprintf("Validated tasks: %d", view.ValidatedTasks),
		fmt.Sprintf("Score: %d%%", view.Score),
		fmt.Sprintf("Time left: %s", remaining),
		renderProgress(view.CurrentIndex, view.Total, view.Completed),
	}
	if view.Completed {
		status = append(status, r.passStyle.Render("Session completed."))
	}
	return r.borderStyle.Render(strings.Join(status, "\n"))
}

func renderProgress(currentIndex, total int, completed bool) string {
	if total == 0 {
		return "[----------] 0%"
	}
	completedTasks := currentIndex
	if completed {
		completedTasks = total
	}
	filled := int(float64(completedTasks) / float64(total) * 10)
	if filled > 10 {
		filled = 10
	}
	return fmt.Sprintf("[%s%s] %d%%", strings.Repeat("#", filled), strings.Repeat("-", 10-filled), completedTasks*100/total)
}

func renderTimer(startedAt time.Time, duration time.Duration) string {
	remaining := duration - time.Since(startedAt)
	if remaining <= 0 {
		return "00:00:00 (time expired)"
	}
	return timerModel{remaining: remaining}.View()
}

func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
