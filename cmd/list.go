package cmd

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/basel5001/k8s-exam-simulator/internal/exam"
	"github.com/spf13/cobra"
)

var knownExams = []string{"cks", "cka", "ckad"}

func newListCmd() *cobra.Command {
	var examFilter string
	var domainFilter string
	var verbose bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available exams and scenarios",
		RunE: func(cmd *cobra.Command, args []string) error {
			manager := mustManager()
			summaries, err := manager.ListScenarios()
			if err != nil {
				return err
			}

			exams := knownExams
			if examFilter != "" {
				examFilter = strings.ToLower(examFilter)
				if !slices.Contains(knownExams, examFilter) {
					return fmt.Errorf("unsupported exam %q; expected one of %s", examFilter, strings.Join(knownExams, ", "))
				}
				exams = []string{examFilter}
				scoped := make([]exam.ScenarioSummary, 0, len(summaries))
				for _, summary := range summaries {
					if summary.Exam == examFilter {
						scoped = append(scoped, summary)
					}
				}
				summaries = scoped
			}

			if domainFilter != "" {
				filtered := make([]exam.ScenarioSummary, 0, len(summaries))
				for _, summary := range summaries {
					if matchesDomain(summary.Domains, domainFilter) {
						filtered = append(filtered, summary)
					}
				}
				if len(filtered) == 0 {
					scope := "any exam"
					if examFilter != "" {
						scope = examFilter
					}
					return fmt.Errorf("no scenarios in %s match domain %q; available domains there:\n  %s", scope, domainFilter, strings.Join(collectDomains(summaries), "\n  "))
				}
				summaries = filtered
			}

			grouped := map[string][]exam.ScenarioSummary{}
			for _, summary := range summaries {
				grouped[summary.Exam] = append(grouped[summary.Exam], summary)
			}

			shown := 0
			for _, name := range exams {
				items := grouped[name]
				if len(items) == 0 {
					continue
				}
				shown++
				sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
				fmt.Printf("%s:\n", name)
				for _, item := range items {
					if verbose {
						printScenarioDetail(item)
						continue
					}
					fmt.Printf("  - %s [%s, %d%%, %dmin]\n", item.Name, item.Difficulty, item.Weight, item.Duration)
				}
				fmt.Printf("  total: %s\n", renderTotals(items))
			}
			if shown > 1 {
				fmt.Printf("\nall exams: %s\n", renderTotals(summaries))
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&examFilter, "exam", "", "Only list scenarios for one exam (cks, cka, ckad)")
	cmd.Flags().StringVar(&domainFilter, "domain", "", "Only list scenarios whose domains match this text (case-insensitive substring)")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Include domains and task descriptions (no hints or solutions)")

	return cmd
}

// renderTotals summarises a scenario set by exam weight and estimated time,
// so a filtered list shows what the revision session is actually worth.
func renderTotals(summaries []exam.ScenarioSummary) string {
	weight := 0
	minutes := 0
	for _, summary := range summaries {
		weight += summary.Weight
		minutes += summary.Duration
	}
	noun := "scenarios"
	if len(summaries) == 1 {
		noun = "scenario"
	}
	return fmt.Sprintf("%d%% across %d %s, ~%s", weight, len(summaries), noun, formatMinutes(minutes))
}

func formatMinutes(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dmin", minutes)
	}
	if minutes%60 == 0 {
		return fmt.Sprintf("%dh", minutes/60)
	}
	return fmt.Sprintf("%dh%02dmin", minutes/60, minutes%60)
}

func matchesDomain(domains []string, filter string) bool {
	needle := strings.ToLower(strings.TrimSpace(filter))
	for _, domain := range domains {
		if strings.Contains(strings.ToLower(domain), needle) {
			return true
		}
	}
	return false
}

func collectDomains(summaries []exam.ScenarioSummary) []string {
	seen := map[string]struct{}{}
	for _, summary := range summaries {
		for _, domain := range summary.Domains {
			seen[domain] = struct{}{}
		}
	}
	domains := make([]string, 0, len(seen))
	for domain := range seen {
		domains = append(domains, domain)
	}
	sort.Strings(domains)
	return domains
}

// printScenarioDetail shows the task text without hints or solutions, so the
// scenario can be previewed without spoiling it.
func printScenarioDetail(summary exam.ScenarioSummary) {
	fmt.Printf("  - %s [%s, %d%%, %dmin]\n", summary.Name, summary.Difficulty, summary.Weight, summary.Duration)
	if len(summary.Domains) > 0 {
		fmt.Printf("      domains: %s\n", strings.Join(summary.Domains, ", "))
	}
	for _, line := range strings.Split(strings.TrimSpace(summary.Description), "\n") {
		fmt.Printf("      %s\n", strings.TrimSpace(line))
	}
	fmt.Println()
}
