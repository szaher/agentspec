package evaluation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// FormatReport formats a RunResult in the specified format.
func FormatReport(result *RunResult, format string) (string, error) {
	switch format {
	case "table":
		return formatTable(result), nil
	case "json":
		return formatJSON(result)
	case "markdown":
		return formatMarkdown(result), nil
	default:
		return "", fmt.Errorf("unsupported format: %q (available: table, json, markdown)", format)
	}
}

// CompareResults compares two evaluation runs and returns a summary.
func CompareResults(current, previous *RunResult) string {
	var b strings.Builder

	b.WriteString("\nCompared to previous run:\n")

	// Overall score change
	diff := current.OverallScore - previous.OverallScore
	if diff > 0 {
		fmt.Fprintf(&b, "  Overall: %.2f → %.2f (+%.2f)\n", previous.OverallScore, current.OverallScore, diff)
	} else if diff < 0 {
		fmt.Fprintf(&b, "  Overall: %.2f → %.2f (%.2f)\n", previous.OverallScore, current.OverallScore, diff)
	} else {
		fmt.Fprintf(&b, "  Overall: %.2f → %.2f (no change)\n", previous.OverallScore, current.OverallScore)
	}

	// Find regressions and improvements
	prevMap := make(map[string]float64)
	for _, c := range previous.Cases {
		prevMap[c.Name] = c.Score
	}

	var regressions, improvements []string
	for _, c := range current.Cases {
		if prevScore, ok := prevMap[c.Name]; ok {
			if c.Score < prevScore-0.01 {
				regressions = append(regressions, fmt.Sprintf("%s: %.2f → %.2f", c.Name, prevScore, c.Score))
			} else if c.Score > prevScore+0.01 {
				improvements = append(improvements, fmt.Sprintf("%s: %.2f → %.2f", c.Name, prevScore, c.Score))
			}
		}
	}

	fmt.Fprintf(&b, "  Regressions: %d\n", len(regressions))
	for _, r := range regressions {
		fmt.Fprintf(&b, "    - %s\n", r)
	}
	fmt.Fprintf(&b, "  Improvements: %d\n", len(improvements))
	for _, i := range improvements {
		fmt.Fprintf(&b, "    + %s\n", i)
	}

	return b.String()
}

func formatTable(result *RunResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Evaluating %s (%d test cases)...\n\n", result.AgentName, result.TotalCases)

	// Find max name length for alignment
	maxLen := 20
	for _, c := range result.Cases {
		if len(c.Name) > maxLen {
			maxLen = len(c.Name)
		}
	}

	for _, c := range result.Cases {
		icon := "✓"
		if !c.Passed {
			icon = "✗"
		}
		padding := strings.Repeat(" ", maxLen-len(c.Name)+2)
		fmt.Fprintf(&b, "  %s %s%sscore: %.2f  (threshold: %.2f)\n",
			icon, c.Name, padding, c.Score, c.Threshold)
		if c.Error != "" {
			fmt.Fprintf(&b, "    error: %s\n", c.Error)
		}
	}

	fmt.Fprintf(&b, "\nResults: %d/%d passed (%d%%)\n",
		result.PassedCases, result.TotalCases,
		percentage(result.PassedCases, result.TotalCases))
	fmt.Fprintf(&b, "Overall score: %.2f\n", result.OverallScore)
	fmt.Fprintf(&b, "Duration: %s\n", result.Duration.Round(time.Millisecond))

	return b.String()
}

func formatJSON(result *RunResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func formatMarkdown(result *RunResult) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# Evaluation Report: %s\n\n", result.AgentName)
	fmt.Fprintf(&b, "**Date**: %s\n", result.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(&b, "**Duration**: %s\n\n", result.Duration.Round(time.Millisecond))

	b.WriteString("## Summary\n\n")
	b.WriteString("| Metric | Value |\n")
	b.WriteString("|----|----|\n")
	fmt.Fprintf(&b, "| Total Cases | %d |\n", result.TotalCases)
	fmt.Fprintf(&b, "| Passed | %d |\n", result.PassedCases)
	fmt.Fprintf(&b, "| Failed | %d |\n", result.FailedCases)
	fmt.Fprintf(&b, "| Pass Rate | %d%% |\n", percentage(result.PassedCases, result.TotalCases))
	fmt.Fprintf(&b, "| Overall Score | %.2f |\n", result.OverallScore)

	b.WriteString("\n## Results\n\n")
	b.WriteString("| Test Case | Score | Threshold | Status | Scoring |\n")
	b.WriteString("|-----------|-------|-----------|--------|--------|\n")
	for _, c := range result.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		fmt.Fprintf(&b, "| %s | %.2f | %.2f | %s | %s |\n",
			c.Name, c.Score, c.Threshold, status, c.Scoring)
	}

	// Show details for failed cases
	hasFailed := false
	for _, c := range result.Cases {
		if !c.Passed {
			if !hasFailed {
				b.WriteString("\n## Failed Cases\n\n")
				hasFailed = true
			}
			fmt.Fprintf(&b, "### %s\n\n", c.Name)
			fmt.Fprintf(&b, "- **Score**: %.2f (threshold: %.2f)\n", c.Score, c.Threshold)
			fmt.Fprintf(&b, "- **Input**: %s\n", c.Input)
			fmt.Fprintf(&b, "- **Expected**: %s\n", c.Expected)
			fmt.Fprintf(&b, "- **Actual**: %s\n", c.Actual)
			if c.Error != "" {
				fmt.Fprintf(&b, "- **Error**: %s\n", c.Error)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func percentage(part, total int) int {
	if total == 0 {
		return 0
	}
	return (part * 100) / total
}
