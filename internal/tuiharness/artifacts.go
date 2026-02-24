package tuiharness

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ArtifactWriter handles saving exploration artifacts.
type ArtifactWriter struct {
	baseDir string
}

// NewArtifactWriter creates a new artifact writer.
func NewArtifactWriter(baseDir string) *ArtifactWriter {
	return &ArtifactWriter{baseDir: baseDir}
}

// SaveFailure saves all artifacts for a failed exploration.
func (a *ArtifactWriter) SaveFailure(result *ExplorationResult) (string, error) {
	// Create timestamped directory
	timestamp := time.Now().Format("20060102_150405")
	dirName := fmt.Sprintf("%s_seed%d", timestamp, result.Seed)
	artifactDir := filepath.Join(a.baseDir, dirName)

	if err := os.MkdirAll(artifactDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %w", err)
	}

	// Save steps.jsonl
	if err := a.saveSteps(artifactDir, result.Steps); err != nil {
		return artifactDir, fmt.Errorf("failed to save steps: %w", err)
	}

	// Save screens
	failureStep := result.FailureStep
	if failureStep >= len(result.Screens) {
		failureStep = len(result.Screens) - 1
	}
	if failureStep < 0 {
		failureStep = 0
	}
	if err := a.saveScreens(artifactDir, result.Screens, failureStep); err != nil {
		return artifactDir, fmt.Errorf("failed to save screens: %w", err)
	}

	// Save repro.txt - handle case where failure is at step 0 with no steps
	stepsToSave := result.Steps
	if result.FailureStep+1 <= len(result.Steps) && result.FailureStep >= 0 {
		stepsToSave = result.Steps[:result.FailureStep+1]
	}
	if err := a.saveRepro(artifactDir, stepsToSave); err != nil {
		return artifactDir, fmt.Errorf("failed to save repro: %w", err)
	}

	// Save failure info
	if err := a.saveFailureInfo(artifactDir, result); err != nil {
		return artifactDir, fmt.Errorf("failed to save failure info: %w", err)
	}

	return artifactDir, nil
}

// saveSteps saves the step history as JSONL.
func (a *ArtifactWriter) saveSteps(dir string, steps []Step) error {
	f, err := os.Create(filepath.Join(dir, "steps.jsonl"))
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	for _, step := range steps {
		record := map[string]any{
			"i":          step.Index,
			"action":     step.Action.String(),
			"screenHash": step.ScreenHash,
			"cursor":     []int{step.CursorX, step.CursorY},
			"size":       []int{step.Cols, step.Rows},
			"newState":   step.NewState,
			"timestamp":  step.Timestamp.Format(time.RFC3339),
		}
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}

	return nil
}

// saveScreens saves the last K screens before failure.
func (a *ArtifactWriter) saveScreens(dir string, screens []*Screen, failureStep int) error {
	screensDir := filepath.Join(dir, "last_k_screens")
	if err := os.MkdirAll(screensDir, 0755); err != nil {
		return err
	}

	// Save last 10 screens (or all if fewer)
	k := 10
	start := failureStep - k + 1
	if start < 0 {
		start = 0
	}
	if failureStep >= len(screens) {
		failureStep = len(screens) - 1
	}

	for i := start; i <= failureStep && i < len(screens); i++ {
		screen := screens[i]

		// Save plain text
		txtPath := filepath.Join(screensDir, fmt.Sprintf("screen_%04d.txt", i))
		if err := os.WriteFile(txtPath, []byte(screen.GridText), 0644); err != nil {
			return err
		}

		// Save ANSI (if available)
		if screen.RawANSI != "" {
			ansiPath := filepath.Join(screensDir, fmt.Sprintf("screen_%04d.ansi", i))
			if err := os.WriteFile(ansiPath, []byte(screen.RawANSI), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// saveRepro saves a replayable reproduction file.
func (a *ArtifactWriter) saveRepro(dir string, steps []Step) error {
	var sb strings.Builder
	sb.WriteString("# TUI Harness Repro File\n")
	sb.WriteString("# Generated: " + time.Now().Format(time.RFC3339) + "\n")
	sb.WriteString("# Steps: " + intToStr(len(steps)) + "\n")
	sb.WriteString("#\n")
	sb.WriteString("# Format: ACTION [ARGS]\n")
	sb.WriteString("# Actions: PRESS <key>, TYPE <text>, RESIZE <width> <height>\n")
	sb.WriteString("#\n\n")

	for _, step := range steps {
		switch step.Action.Type {
		case ActionKeyPress:
			sb.WriteString("PRESS " + step.Action.Key.String() + "\n")
		case ActionType_:
			sb.WriteString("TYPE " + step.Action.Text + "\n")
		case ActionResize:
			sb.WriteString(fmt.Sprintf("RESIZE %d %d\n", step.Action.Width, step.Action.Height))
		}
	}

	return os.WriteFile(filepath.Join(dir, "repro.txt"), []byte(sb.String()), 0644)
}

// saveFailureInfo saves information about the failure.
func (a *ArtifactWriter) saveFailureInfo(dir string, result *ExplorationResult) error {
	info := map[string]any{
		"seed":         result.Seed,
		"totalSteps":   result.TotalSteps,
		"failureStep":  result.FailureStep,
		"uniqueStates": result.UniqueStates,
		"duration":     result.Duration.String(),
		"stuckEvents":  result.StuckEvents,
	}

	if result.Failure != nil {
		info["oracle"] = result.Failure.OracleName
		info["description"] = result.Failure.Description
		info["details"] = result.Failure.Details
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, "failure_info.json"), data, 0644)
}

// ParseRepro parses a repro file and returns the actions.
func ParseRepro(path string) ([]Action, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var actions []Action
	lines := strings.Split(string(data), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 0 {
			continue
		}

		cmd := parts[0]
		args := ""
		if len(parts) > 1 {
			args = parts[1]
		}

		switch cmd {
		case "PRESS":
			key := parseKeyName(args)
			actions = append(actions, PressAction(key))
		case "TYPE":
			actions = append(actions, TypeAction(args))
		case "RESIZE":
			var w, h int
			fmt.Sscanf(args, "%d %d", &w, &h)
			actions = append(actions, ResizeAction(w, h))
		}
	}

	return actions, nil
}

// parseKeyName converts a key name string to a Key.
func parseKeyName(name string) Key {
	switch name {
	case "Up":
		return KeyUp
	case "Down":
		return KeyDown
	case "Left":
		return KeyLeft
	case "Right":
		return KeyRight
	case "Enter":
		return KeyEnter
	case "Esc":
		return KeyEsc
	case "Tab":
		return KeyTab
	case "Shift+Tab":
		return KeyShiftTab
	case "Backspace":
		return KeyBackspace
	case "Space":
		return KeySpace
	case "Ctrl+C":
		return KeyCtrlC
	case "Ctrl+R":
		return KeyCtrlR
	case "Ctrl+A":
		return KeyCtrlA
	case "Ctrl+E":
		return KeyCtrlE
	case "PgUp":
		return KeyPgUp
	case "PgDn":
		return KeyPgDown
	case "Home":
		return KeyHome
	case "End":
		return KeyEnd
	default:
		// Assume single character
		if len(name) == 1 {
			return Rune(rune(name[0]))
		}
		return KeyEnter // fallback
	}
}

// Replay replays a list of actions on a harness and returns the result.
func Replay(h *Harness, actions []Action, oracles *CompositeOracle, waitTime time.Duration) (*ExplorationResult, error) {
	result := &ExplorationResult{
		Steps:        make([]Step, 0, len(actions)),
		Screens:      make([]*Screen, 0, len(actions)+1),
		ActionCounts: make(map[string]int),
	}
	startTime := time.Now()

	// Get initial screen
	screen, err := h.Screen()
	if err != nil {
		return nil, err
	}
	result.Screens = append(result.Screens, screen)

	// Check initial screen
	if oracleResult := oracles.Check(screen, result.Screens); oracleResult != nil && oracleResult.Failed {
		result.Failure = oracleResult
		result.FailureStep = 0
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Replay actions
	for i, action := range actions {
		result.ActionCounts[action.String()]++

		// Execute action
		switch action.Type {
		case ActionKeyPress:
			if err := h.Press(action.Key); err != nil {
				return result, err
			}
		case ActionType_:
			if err := h.Type(action.Text); err != nil {
				return result, err
			}
		case ActionResize:
			if err := h.Resize(action.Width, action.Height); err != nil {
				return result, err
			}
		}

		// Wait
		time.Sleep(waitTime)

		// Get screen
		newScreen, err := h.Screen()
		if err != nil {
			return result, err
		}

		// Record step
		step := Step{
			Index:      i,
			Action:     action,
			ScreenHash: newScreen.Hash(),
			CursorX:    newScreen.CursorX,
			CursorY:    newScreen.CursorY,
			Cols:       newScreen.Cols,
			Rows:       newScreen.Rows,
			Timestamp:  time.Now(),
		}
		result.Steps = append(result.Steps, step)
		result.Screens = append(result.Screens, newScreen)

		// Check oracles
		if oracleResult := oracles.Check(newScreen, result.Screens); oracleResult != nil && oracleResult.Failed {
			result.Failure = oracleResult
			result.FailureStep = i
			result.Duration = time.Since(startTime)
			result.TotalSteps = i + 1
			return result, nil
		}
	}

	result.Duration = time.Since(startTime)
	result.TotalSteps = len(actions)
	return result, nil
}
