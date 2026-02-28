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
		b.WriteString(fmt.Sprintf("  Overall: %.2f → %.2f (+%.2f)\n", previous.OverallScore, current.OverallScore, diff))
	} else if diff < 0 {
		b.WriteString(fmt.Sprintf("  Overall: %.2f → %.2f (%.2f)\n", previous.OverallScore, current.OverallScore, diff))
	} else {
		b.WriteString(fmt.Sprintf("  Overall: %.2f → %.2f (no change)\n", previous.OverallScore, current.OverallScore))
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

	b.WriteString(fmt.Sprintf("  Regressions: %d\n", len(regressions)))
	for _, r := range regressions {
		b.WriteString(fmt.Sprintf("    - %s\n", r))
	}
	b.WriteString(fmt.Sprintf("  Improvements: %d\n", len(improvements)))
	for _, i := range improvements {
		b.WriteString(fmt.Sprintf("    + %s\n", i))
	}

	return b.String()
}

func formatTable(result *RunResult) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Evaluating %s (%d test cases)...\n\n", result.AgentName, result.TotalCases))

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
		b.WriteString(fmt.Sprintf("  %s %s%sscore: %.2f  (threshold: %.2f)\n",
			icon, c.Name, padding, c.Score, c.Threshold))
		if c.Error != "" {
			b.WriteString(fmt.Sprintf("    error: %s\n", c.Error))
		}
	}

	b.WriteString(fmt.Sprintf("\nResults: %d/%d passed (%d%%)\n",
		result.PassedCases, result.TotalCases,
		percentage(result.PassedCases, result.TotalCases)))
	b.WriteString(fmt.Sprintf("Overall score: %.2f\n", result.OverallScore))
	b.WriteString(fmt.Sprintf("Duration: %s\n", result.Duration.Round(time.Millisecond)))

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

	b.WriteString(fmt.Sprintf("# Evaluation Report: %s\n\n", result.AgentName))
	b.WriteString(fmt.Sprintf("**Date**: %s\n", result.Timestamp.Format("2006-01-02 15:04:05")))
	b.WriteString(fmt.Sprintf("**Duration**: %s\n\n", result.Duration.Round(time.Millisecond)))

	b.WriteString("## Summary\n\n")
	b.WriteString(fmt.Sprintf("| Metric | Value |\n"))
	b.WriteString("|----|----|\n")
	b.WriteString(fmt.Sprintf("| Total Cases | %d |\n", result.TotalCases))
	b.WriteString(fmt.Sprintf("| Passed | %d |\n", result.PassedCases))
	b.WriteString(fmt.Sprintf("| Failed | %d |\n", result.FailedCases))
	b.WriteString(fmt.Sprintf("| Pass Rate | %d%% |\n", percentage(result.PassedCases, result.TotalCases)))
	b.WriteString(fmt.Sprintf("| Overall Score | %.2f |\n", result.OverallScore))

	b.WriteString("\n## Results\n\n")
	b.WriteString("| Test Case | Score | Threshold | Status | Scoring |\n")
	b.WriteString("|-----------|-------|-----------|--------|--------|\n")
	for _, c := range result.Cases {
		status := "PASS"
		if !c.Passed {
			status = "FAIL"
		}
		b.WriteString(fmt.Sprintf("| %s | %.2f | %.2f | %s | %s |\n",
			c.Name, c.Score, c.Threshold, status, c.Scoring))
	}

	// Show details for failed cases
	hasFailed := false
	for _, c := range result.Cases {
		if !c.Passed {
			if !hasFailed {
				b.WriteString("\n## Failed Cases\n\n")
				hasFailed = true
			}
			b.WriteString(fmt.Sprintf("### %s\n\n", c.Name))
			b.WriteString(fmt.Sprintf("- **Score**: %.2f (threshold: %.2f)\n", c.Score, c.Threshold))
			b.WriteString(fmt.Sprintf("- **Input**: %s\n", c.Input))
			b.WriteString(fmt.Sprintf("- **Expected**: %s\n", c.Expected))
			b.WriteString(fmt.Sprintf("- **Actual**: %s\n", c.Actual))
			if c.Error != "" {
				b.WriteString(fmt.Sprintf("- **Error**: %s\n", c.Error))
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
