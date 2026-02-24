package tuiharness

import (
	"os"
	"path/filepath"
	"time"
)

// Minimizer performs delta-debugging style minimization of failure repros.
type Minimizer struct {
	factory  ModelFactory
	oracles  *CompositeOracle
	opts     Options
	waitTime time.Duration
	maxRuns  int
}

// NewMinimizer creates a new minimizer.
func NewMinimizer(factory ModelFactory, oracles *CompositeOracle, opts Options) *Minimizer {
	return &Minimizer{
		factory:  factory,
		oracles:  oracles,
		opts:     opts,
		waitTime: 50 * time.Millisecond,
		maxRuns:  100, // Limit minimization attempts
	}
}

// Minimize attempts to shrink a failing action sequence.
// Returns the minimized sequence and whether minimization was successful.
func (m *Minimizer) Minimize(actions []Action) ([]Action, bool) {
	if len(actions) <= 1 {
		return actions, false
	}

	// Verify the original sequence still fails
	if !m.reproduces(actions) {
		return actions, false
	}

	current := actions
	runs := 0

	// Try progressively smaller chunks
	for chunkSize := len(current) / 2; chunkSize >= 1 && runs < m.maxRuns; chunkSize = max(1, chunkSize/2) {
		improved := true
		for improved && runs < m.maxRuns {
			improved = false

			// Try removing each chunk
			for i := 0; i <= len(current)-chunkSize && runs < m.maxRuns; i++ {
				runs++

				// Create sequence without chunk [i:i+chunkSize]
				candidate := make([]Action, 0, len(current)-chunkSize)
				candidate = append(candidate, current[:i]...)
				candidate = append(candidate, current[i+chunkSize:]...)

				if len(candidate) == 0 {
					continue
				}

				if m.reproduces(candidate) {
					current = candidate
					improved = true
					break // Restart with new sequence
				}
			}
		}

		if chunkSize == 1 {
			break
		}
	}

	return current, len(current) < len(actions)
}

// reproduces checks if the action sequence triggers a failure.
func (m *Minimizer) reproduces(actions []Action) bool {
	h, err := Start(m.factory, m.opts)
	if err != nil {
		return false
	}
	defer h.Stop()

	result, err := Replay(h, actions, m.oracles, m.waitTime)
	if err != nil {
		return false
	}

	return result.Failure != nil
}

// MinimizeAndSave minimizes a failure repro and saves both versions.
func (m *Minimizer) MinimizeAndSave(artifactDir string, originalSteps []Step) error {
	// Extract actions from steps
	actions := make([]Action, len(originalSteps))
	for i, step := range originalSteps {
		actions[i] = step.Action
	}

	// Minimize
	minimized, improved := m.Minimize(actions)
	if !improved {
		return nil // Nothing to save
	}

	// Save minimized repro
	return m.saveMinimizedRepro(artifactDir, minimized)
}

// saveMinimizedRepro saves the minimized reproduction file.
func (m *Minimizer) saveMinimizedRepro(dir string, actions []Action) error {
	var sb stringBuilder
	sb.WriteString("# Minimized TUI Harness Repro File\n")
	sb.WriteString("# Generated: " + time.Now().Format(time.RFC3339) + "\n")
	sb.WriteString("# Steps: " + intToStr(len(actions)) + "\n")
	sb.WriteString("#\n")
	sb.WriteString("# Format: ACTION [ARGS]\n")
	sb.WriteString("#\n\n")

	for _, action := range actions {
		switch action.Type {
		case ActionKeyPress:
			sb.WriteString("PRESS " + action.Key.String() + "\n")
		case ActionType_:
			sb.WriteString("TYPE " + action.Text + "\n")
		case ActionResize:
			sb.WriteString("RESIZE " + intToStr(action.Width) + " " + intToStr(action.Height) + "\n")
		}
	}

	return os.WriteFile(filepath.Join(dir, "repro_minimized.txt"), []byte(sb.String()), 0644)
}

// stringBuilder is a simple string builder to avoid importing strings.
type stringBuilder struct {
	data []byte
}

func (s *stringBuilder) WriteString(str string) {
	s.data = append(s.data, []byte(str)...)
}

func (s *stringBuilder) String() string {
	return string(s.data)
}

// MinimizationResult contains the result of minimization.
type MinimizationResult struct {
	OriginalLength   int
	MinimizedLength  int
	Improved         bool
	MinimizedActions []Action
	Attempts         int
}

// MinimizeWithStats performs minimization and returns statistics.
func (m *Minimizer) MinimizeWithStats(actions []Action) MinimizationResult {
	result := MinimizationResult{
		OriginalLength: len(actions),
	}

	if len(actions) <= 1 {
		result.MinimizedActions = actions
		result.MinimizedLength = len(actions)
		return result
	}

	// Verify original fails
	if !m.reproduces(actions) {
		result.MinimizedActions = actions
		result.MinimizedLength = len(actions)
		return result
	}

	current := actions
	attempts := 0

	// Binary-search style reduction
	for chunkSize := len(current) / 2; chunkSize >= 1 && attempts < m.maxRuns; {
		improved := false

		for i := 0; i <= len(current)-chunkSize && attempts < m.maxRuns; i++ {
			attempts++

			candidate := make([]Action, 0, len(current)-chunkSize)
			candidate = append(candidate, current[:i]...)
			candidate = append(candidate, current[i+chunkSize:]...)

			if len(candidate) == 0 {
				continue
			}

			if m.reproduces(candidate) {
				current = candidate
				improved = true
				break
			}
		}

		if !improved {
			chunkSize /= 2
		}
	}

	result.MinimizedActions = current
	result.MinimizedLength = len(current)
	result.Improved = len(current) < len(actions)
	result.Attempts = attempts
	return result
}
