package exam

import "time"

type Scenario struct {
	Name        string   `yaml:"name"`
	Exam        string   `yaml:"exam"`
	Weight      int      `yaml:"weight"`
	Difficulty  string   `yaml:"difficulty"`
	Duration    int      `yaml:"duration"`
	Domains     []string `yaml:"domains"`
	Description string   `yaml:"description"`
	Context     string   `yaml:"context"`
	Setup       string   `yaml:"setup"`
	Hints       []string `yaml:"hints"`
	Validation  string   `yaml:"validation"`
	Solution    string   `yaml:"solution"`
}

type ScenarioSummary struct {
	Name        string
	Exam        string
	Weight      int
	Difficulty  string
	Duration    int
	Domains     []string
	Description string
}

type Session struct {
	ExamType      string          `yaml:"exam_type"`
	ClusterName   string          `yaml:"cluster_name"`
	ScenarioNames []string        `yaml:"scenario_names"`
	CurrentIndex  int             `yaml:"current_index"`
	StartedAt     time.Time       `yaml:"started_at"`
	Duration      time.Duration   `yaml:"duration"`
	NoTimer       bool            `yaml:"no_timer"`
	PracticeMode  bool            `yaml:"practice_mode"`
	Score         int             `yaml:"score"`
	TotalWeight   int             `yaml:"total_weight"`
	Results       map[string]bool `yaml:"results"`
	HintIndexes   map[string]int  `yaml:"hint_indexes"`
	CompletedAt   *time.Time      `yaml:"completed_at,omitempty"`
}

type ValidationResult struct {
	Passed   bool
	Output   string
	Scenario Scenario
}
