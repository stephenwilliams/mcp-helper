package tuiharness

import (
	"math/rand"
	"time"
)

// ExplorerConfig configures the autonomous exploration.
type ExplorerConfig struct {
	Seed            int64         // Random seed for reproducibility
	MaxSteps        int           // Maximum steps per episode
	StuckThreshold  int           // Actions without screen change before "stuck"
	ActionSpace     []Action      // Available actions
	TextActions     []Action      // Text input actions (used occasionally)
	TextProbability float64       // Probability of choosing text action (0-1)
	ResizeProbability float64     // Probability of resize (0-1)
	WaitAfterAction time.Duration // Wait time after each action
	EnableMinimize  bool          // Attempt to minimize failure repros
}

// DefaultExplorerConfig returns default exploration settings.
func DefaultExplorerConfig() ExplorerConfig {
	return ExplorerConfig{
		Seed:              time.Now().UnixNano(),
		MaxSteps:          500,
		StuckThreshold:    5,
		ActionSpace:       DefaultActionSpace(),
		TextActions:       RandomTextActions(),
		TextProbability:   0.05,
		ResizeProbability: 0.02,
		WaitAfterAction:   50 * time.Millisecond,
		EnableMinimize:    true,
	}
}

// Explorer performs autonomous exploration of a TUI.
type Explorer struct {
	config       ExplorerConfig
	rng          *rand.Rand
	visitedHashes map[string]int // hash -> visit count
	history      []Step
	screens      []*Screen
	stuckCount   int
	totalSteps   int
}

// Step records a single exploration step.
type Step struct {
	Index      int
	Action     Action
	ScreenHash string
	CursorX    int
	CursorY    int
	Cols       int
	Rows       int
	Timestamp  time.Time
	NewState   bool // True if this action reached a previously unseen state
}

// ExplorationResult contains the results of an exploration run.
type ExplorationResult struct {
	Seed          int64
	TotalSteps    int
	UniqueStates  int
	Steps         []Step
	Screens       []*Screen
	Failure       *OracleResult
	FailureStep   int
	Duration      time.Duration
	ActionCounts  map[string]int
	StuckEvents   int
}

// NewExplorer creates a new explorer with the given config.
func NewExplorer(config ExplorerConfig) *Explorer {
	return &Explorer{
		config:        config,
		rng:           rand.New(rand.NewSource(config.Seed)),
		visitedHashes: make(map[string]int),
		history:       make([]Step, 0, config.MaxSteps),
		screens:       make([]*Screen, 0, config.MaxSteps),
	}
}

// Run performs exploration on the given harness.
func (e *Explorer) Run(h *Harness, oracles *CompositeOracle) *ExplorationResult {
	startTime := time.Now()
	result := &ExplorationResult{
		Seed:         e.config.Seed,
		ActionCounts: make(map[string]int),
	}

	// Wait for initial render
	time.Sleep(100 * time.Millisecond)

	// Get initial screen
	screen, err := h.Screen()
	if err != nil {
		return result
	}
	e.recordScreen(screen)

	// Skip oracle check on initial screen - it may not be fully rendered yet
	// Oracles will be checked after the first action

	// Exploration loop
	for step := 0; step < e.config.MaxSteps; step++ {
		// Select action
		action := e.selectAction(screen)
		result.ActionCounts[action.String()]++

		// Execute action
		if err := e.executeAction(h, action); err != nil {
			break // Harness stopped
		}

		// Wait for screen to settle
		time.Sleep(e.config.WaitAfterAction)

		// Get new screen
		prevHash := screen.Hash()
		newScreen, err := h.Screen()
		if err != nil {
			break
		}

		// Record step
		isNew := e.visitedHashes[newScreen.Hash()] == 0
		stepRecord := Step{
			Index:      step,
			Action:     action,
			ScreenHash: newScreen.Hash(),
			CursorX:    newScreen.CursorX,
			CursorY:    newScreen.CursorY,
			Cols:       newScreen.Cols,
			Rows:       newScreen.Rows,
			Timestamp:  time.Now(),
			NewState:   isNew,
		}
		e.history = append(e.history, stepRecord)
		e.recordScreen(newScreen)

		// Check for stuck state
		if newScreen.Hash() == prevHash {
			e.stuckCount++
			if e.stuckCount >= e.config.StuckThreshold {
				result.StuckEvents++
				e.tryUnstuck(h)
			}
		} else {
			e.stuckCount = 0
		}

		// Run oracles
		if oracleResult := oracles.Check(newScreen, e.screens); oracleResult != nil && oracleResult.Failed {
			result.Failure = oracleResult
			result.FailureStep = step
			result.Steps = e.history
			result.Screens = e.screens
			result.Duration = time.Since(startTime)
			result.TotalSteps = step + 1
			result.UniqueStates = len(e.visitedHashes)
			return result
		}

		screen = newScreen
		e.totalSteps++
	}

	result.Steps = e.history
	result.Screens = e.screens
	result.Duration = time.Since(startTime)
	result.TotalSteps = e.totalSteps
	result.UniqueStates = len(e.visitedHashes)
	return result
}

// selectAction chooses the next action using novelty guidance.
func (e *Explorer) selectAction(currentScreen *Screen) Action {
	// Occasionally use text input
	if e.rng.Float64() < e.config.TextProbability && len(e.config.TextActions) > 0 {
		return e.config.TextActions[e.rng.Intn(len(e.config.TextActions))]
	}

	// Occasionally resize
	if e.rng.Float64() < e.config.ResizeProbability {
		resizeActions := []Action{
			ResizeAction(80, 24),
			ResizeAction(100, 30),
			ResizeAction(120, 40),
		}
		return resizeActions[e.rng.Intn(len(resizeActions))]
	}

	// Use novelty-guided selection
	// Give preference to actions that might lead to new states
	// This is a simple heuristic: weight less-used actions higher

	actionWeights := make([]float64, len(e.config.ActionSpace))
	totalWeight := 0.0

	for i, action := range e.config.ActionSpace {
		// Base weight
		weight := 1.0

		// Reduce weight for frequently used actions
		count := e.history[max(0, len(e.history)-20):]
		recentUses := 0
		for _, step := range count {
			if step.Action.String() == action.String() {
				recentUses++
			}
		}
		weight /= float64(1 + recentUses)

		actionWeights[i] = weight
		totalWeight += weight
	}

	// Weighted random selection
	r := e.rng.Float64() * totalWeight
	cumulative := 0.0
	for i, weight := range actionWeights {
		cumulative += weight
		if r <= cumulative {
			return e.config.ActionSpace[i]
		}
	}

	// Fallback
	return e.config.ActionSpace[e.rng.Intn(len(e.config.ActionSpace))]
}

// executeAction performs the given action on the harness.
func (e *Explorer) executeAction(h *Harness, action Action) error {
	switch action.Type {
	case ActionKeyPress:
		return h.Press(action.Key)
	case ActionType_:
		return h.Type(action.Text)
	case ActionResize:
		return h.Resize(action.Width, action.Height)
	}
	return nil
}

// recordScreen records a screen snapshot.
func (e *Explorer) recordScreen(screen *Screen) {
	e.screens = append(e.screens, screen)
	e.visitedHashes[screen.Hash()]++
}

// tryUnstuck attempts to escape a stuck state.
func (e *Explorer) tryUnstuck(h *Harness) {
	// Try escape sequences in order
	unstuckActions := []Action{
		PressAction(KeyEsc),
		PressAction(KeyCtrlC),
		PressAction(KeyEnter),
		PressAction(KeyTab),
		ResizeAction(120, 40),
	}

	for _, action := range unstuckActions {
		_ = e.executeAction(h, action)
		time.Sleep(100 * time.Millisecond)

		screen, err := h.Screen()
		if err != nil {
			return
		}

		// If screen changed, we're unstuck
		if len(e.screens) > 0 && screen.Hash() != e.screens[len(e.screens)-1].Hash() {
			e.stuckCount = 0
			return
		}
	}
}

// GetHistory returns the exploration history.
func (e *Explorer) GetHistory() []Step {
	return e.history
}

// GetScreens returns captured screens.
func (e *Explorer) GetScreens() []*Screen {
	return e.screens
}

// GetUniqueStateCount returns the number of unique states visited.
func (e *Explorer) GetUniqueStateCount() int {
	return len(e.visitedHashes)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
