package tuiharness

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// Report generates human-readable exploration reports.
type Report struct {
	results       []*ExplorationResult
	artifactPaths map[int64]string // seed -> artifact path
}

// NewReport creates a new report.
func NewReport() *Report {
	return &Report{
		results:       make([]*ExplorationResult, 0),
		artifactPaths: make(map[int64]string),
	}
}

// AddResult adds an exploration result to the report.
func (r *Report) AddResult(result *ExplorationResult, artifactPath string) {
	r.results = append(r.results, result)
	if artifactPath != "" {
		r.artifactPaths[result.Seed] = artifactPath
	}
}

// Write writes the report to the given writer.
func (r *Report) Write(w io.Writer) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "                    TUI EXPLORATION REPORT")
	fmt.Fprintln(w, "═══════════════════════════════════════════════════════════════")
	fmt.Fprintln(w, "")

	// Summary statistics
	r.writeSummary(w)

	// Failures
	r.writeFailures(w)

	// Coverage stats
	r.writeCoverage(w)

	// Action distribution
	r.writeActionDistribution(w)

	// Near-fail signals
	r.writeNearFailSignals(w)
}

// writeSummary writes the summary section.
func (r *Report) writeSummary(w io.Writer) {
	fmt.Fprintln(w, "SUMMARY")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────")

	totalEpisodes := len(r.results)
	totalSteps := 0
	totalUniqueStates := 0
	failureCount := 0
	totalDuration := int64(0)

	uniqueStatesSet := make(map[string]bool)

	for _, result := range r.results {
		totalSteps += result.TotalSteps
		if result.Failure != nil {
			failureCount++
		}
		totalDuration += result.Duration.Milliseconds()

		// Collect unique states across all runs
		for _, screen := range result.Screens {
			uniqueStatesSet[screen.Hash()] = true
		}
	}
	totalUniqueStates = len(uniqueStatesSet)

	fmt.Fprintf(w, "  Episodes run:      %d\n", totalEpisodes)
	fmt.Fprintf(w, "  Total steps:       %d\n", totalSteps)
	fmt.Fprintf(w, "  Unique states:     %d\n", totalUniqueStates)
	fmt.Fprintf(w, "  Failures found:    %d\n", failureCount)
	fmt.Fprintf(w, "  Total duration:    %dms\n", totalDuration)
	fmt.Fprintln(w, "")

	if failureCount == 0 {
		fmt.Fprintln(w, "  ✓ No bugs detected")
	} else {
		fmt.Fprintf(w, "  ✗ %d bug(s) detected\n", failureCount)
	}
	fmt.Fprintln(w, "")
}

// writeFailures writes details about each failure.
func (r *Report) writeFailures(w io.Writer) {
	failures := r.getFailures()
	if len(failures) == 0 {
		return
	}

	fmt.Fprintln(w, "FAILURES")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────")

	for i, result := range failures {
		fmt.Fprintf(w, "\n  [%d] %s\n", i+1, result.Failure.OracleName)
		fmt.Fprintf(w, "      Description: %s\n", result.Failure.Description)
		if result.Failure.Details != "" {
			fmt.Fprintf(w, "      Details:     %s\n", result.Failure.Details)
		}
		fmt.Fprintf(w, "      Step:        %d of %d\n", result.FailureStep, result.TotalSteps)
		fmt.Fprintf(w, "      Seed:        %d\n", result.Seed)

		if path, ok := r.artifactPaths[result.Seed]; ok {
			fmt.Fprintf(w, "      Artifacts:   %s\n", path)
			fmt.Fprintf(w, "      Repro:       go test -run TestReplayRepro -repro %s/repro.txt\n", path)
		}
	}
	fmt.Fprintln(w, "")
}

// writeCoverage writes coverage statistics.
func (r *Report) writeCoverage(w io.Writer) {
	fmt.Fprintln(w, "COVERAGE")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────")

	// Aggregate unique states
	stateVisits := make(map[string]int)
	for _, result := range r.results {
		for _, screen := range result.Screens {
			stateVisits[screen.Hash()]++
		}
	}

	// Calculate statistics
	totalStates := len(stateVisits)
	singleVisit := 0
	maxVisits := 0

	for _, count := range stateVisits {
		if count == 1 {
			singleVisit++
		}
		if count > maxVisits {
			maxVisits = count
		}
	}

	fmt.Fprintf(w, "  Total unique states:    %d\n", totalStates)
	fmt.Fprintf(w, "  Single-visit states:    %d (%.1f%%)\n",
		singleVisit, 100*float64(singleVisit)/float64(max(1, totalStates)))
	fmt.Fprintf(w, "  Most visited state:     %d times\n", maxVisits)

	// New state discovery rate
	if len(r.results) > 0 {
		newStatesPerEpisode := float64(totalStates) / float64(len(r.results))
		fmt.Fprintf(w, "  New states/episode:     %.1f\n", newStatesPerEpisode)
	}
	fmt.Fprintln(w, "")
}

// writeActionDistribution writes action usage statistics.
func (r *Report) writeActionDistribution(w io.Writer) {
	fmt.Fprintln(w, "ACTION DISTRIBUTION")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────")

	// Aggregate action counts
	actionCounts := make(map[string]int)
	for _, result := range r.results {
		for action, count := range result.ActionCounts {
			actionCounts[action] += count
		}
	}

	// Sort by count
	type actionCount struct {
		action string
		count  int
	}
	sorted := make([]actionCount, 0, len(actionCounts))
	for action, count := range actionCounts {
		sorted = append(sorted, actionCount{action, count})
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].count > sorted[j].count
	})

	// Calculate total
	total := 0
	for _, ac := range sorted {
		total += ac.count
	}

	// Print top actions
	maxShow := 15
	if len(sorted) < maxShow {
		maxShow = len(sorted)
	}

	for i := 0; i < maxShow; i++ {
		ac := sorted[i]
		pct := 100 * float64(ac.count) / float64(max(1, total))
		bar := strings.Repeat("█", int(pct/5))
		fmt.Fprintf(w, "  %-20s %5d (%5.1f%%) %s\n", ac.action, ac.count, pct, bar)
	}

	if len(sorted) > maxShow {
		fmt.Fprintf(w, "  ... and %d more actions\n", len(sorted)-maxShow)
	}
	fmt.Fprintln(w, "")
}

// writeNearFailSignals writes information about near-failure events.
func (r *Report) writeNearFailSignals(w io.Writer) {
	fmt.Fprintln(w, "NEAR-FAIL SIGNALS")
	fmt.Fprintln(w, "───────────────────────────────────────────────────────────────")

	// Collect stuck events
	totalStuck := 0
	for _, result := range r.results {
		totalStuck += result.StuckEvents
	}

	fmt.Fprintf(w, "  Stuck events:           %d\n", totalStuck)

	// Could add more near-fail signals here in the future

	if totalStuck == 0 {
		fmt.Fprintln(w, "  No significant near-fail signals detected")
	}
	fmt.Fprintln(w, "")
}

// getFailures returns all results that have failures.
func (r *Report) getFailures() []*ExplorationResult {
	var failures []*ExplorationResult
	for _, result := range r.results {
		if result.Failure != nil {
			failures = append(failures, result)
		}
	}
	return failures
}

// HasFailures returns true if any failures were found.
func (r *Report) HasFailures() bool {
	for _, result := range r.results {
		if result.Failure != nil {
			return true
		}
	}
	return false
}

// FailureCount returns the number of failures.
func (r *Report) FailureCount() int {
	count := 0
	for _, result := range r.results {
		if result.Failure != nil {
			count++
		}
	}
	return count
}

// String returns the report as a string.
func (r *Report) String() string {
	var sb strings.Builder
	r.Write(&sb)
	return sb.String()
}
