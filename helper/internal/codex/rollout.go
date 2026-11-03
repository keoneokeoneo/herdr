// Package codex reads Codex session logs (rollouts) and runs Codex under
// herdr-poke, mirroring what the log says into terminal labels and limits.json.
package codex

import (
	"bytes"
	"encoding/json"
	"math"
	"time"

	"herdr-poke/internal/state"
)

// baselineTokens is the prompt Codex always carries; its own "% left" formula
// leaves it out of both the window and the usage.
const baselineTokens = 12000

// Snapshot is what the rollout lines seen so far say about a session.
type Snapshot struct {
	Model, Effort string
	CtxPct        int // context used, 0-100; meaningful only if HasCtx
	HasCtx        bool
	Limits        state.ProviderLimits // meaningful only if HasLimits
	HasLimits     bool

	window int // model_context_window from task_started, for token_count without one
}

// CtxPct is how much of a Codex context window is used, as Codex counts it:
// 100 minus the percent left of (window - 12000) after (tokens - 12000).
func CtxPct(window, tokens int) int {
	if window <= baselineTokens {
		return 100
	}
	effective := float64(window - baselineTokens)
	used := math.Max(0, float64(tokens-baselineTokens))
	left := math.Min(100, math.Max(0, 100*(effective-used)/effective))
	return 100 - int(math.Round(left))
}

// Substrings every line Apply cares about contains. Checking them first skips
// the JSON decode of the (often huge) response items and tool outputs.
var (
	markTurnContext = []byte(`"turn_context"`)
	markTokenCount  = []byte(`"token_count"`)
	markTaskStarted = []byte(`"task_started"`)
)

type rolloutLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type turnContext struct {
	Model  string `json:"model"`
	Effort string `json:"effort"`
}

type eventHead struct {
	Type               string `json:"type"`
	ModelContextWindow *int   `json:"model_context_window"`
}

type tokenCount struct {
	Info *struct {
		LastTokenUsage *struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"last_token_usage"`
		ModelContextWindow *int `json:"model_context_window"`
	} `json:"info"`
	RateLimits *struct {
		Primary   *rateWindow `json:"primary"`
		Secondary *rateWindow `json:"secondary"`
		PlanType  string      `json:"plan_type"`
	} `json:"rate_limits"`
}

type rateWindow struct {
	UsedPercent   *float64 `json:"used_percent"`
	WindowMinutes *int     `json:"window_minutes"`
	ResetsAt      *int64   `json:"resets_at"`
}

// Apply updates s from one rollout JSONL line. Lines it does not understand
// (including malformed ones) leave s untouched.
func Apply(s *Snapshot, line []byte) {
	if !bytes.Contains(line, markTurnContext) && !bytes.Contains(line, markTokenCount) &&
		!bytes.Contains(line, markTaskStarted) {
		return
	}
	var l rolloutLine
	if json.Unmarshal(line, &l) != nil {
		return
	}
	switch l.Type {
	case "turn_context":
		var tc turnContext
		if json.Unmarshal(l.Payload, &tc) != nil {
			return
		}
		if tc.Model != "" {
			s.Model = tc.Model
		}
		if tc.Effort != "" {
			s.Effort = tc.Effort
		}
	case "event_msg":
		var head eventHead
		if json.Unmarshal(l.Payload, &head) != nil {
			return
		}
		switch head.Type {
		case "task_started":
			if head.ModelContextWindow != nil && *head.ModelContextWindow > 0 {
				s.window = *head.ModelContextWindow
			}
		case "token_count":
			var tc tokenCount
			if json.Unmarshal(l.Payload, &tc) != nil {
				return
			}
			s.applyTokenCount(tc, l.Timestamp)
		}
	}
}

func (s *Snapshot) applyTokenCount(tc tokenCount, timestamp string) {
	if info := tc.Info; info != nil && info.LastTokenUsage != nil {
		window := s.window
		if info.ModelContextWindow != nil && *info.ModelContextWindow > 0 {
			window = *info.ModelContextWindow
		}
		if window > 0 {
			s.CtxPct = CtxPct(window, info.LastTokenUsage.TotalTokens)
			s.HasCtx = true
		}
	}

	rl := tc.RateLimits
	if rl == nil {
		return
	}
	var ws []state.Window
	for _, w := range []*rateWindow{rl.Primary, rl.Secondary} {
		if w == nil || w.UsedPercent == nil || w.WindowMinutes == nil || w.ResetsAt == nil {
			continue
		}
		ws = append(ws, state.Window{Minutes: *w.WindowMinutes, Used: *w.UsedPercent, ResetsAt: *w.ResetsAt})
	}
	if len(ws) == 0 {
		return
	}
	updated := int64(0)
	if t, err := time.Parse(time.RFC3339Nano, timestamp); err == nil {
		updated = t.Unix()
	}
	s.Limits = state.ProviderLimits{Plan: rl.PlanType, Windows: ws, Updated: updated}
	s.HasLimits = true
}

// Meta is the session_meta line at the top of a rollout.
type Meta struct {
	ID, Cwd, Originator string
	ParentThreadID      string // set for sub-agent threads
	ForkedFromID        string // set for forked threads
	Timestamp           time.Time
}

type metaPayload struct {
	ID             string          `json:"id"`
	Cwd            string          `json:"cwd"`
	Originator     string          `json:"originator"`
	Timestamp      string          `json:"timestamp"`
	ParentThreadID string          `json:"parent_thread_id"`
	ForkedFromID   string          `json:"forked_from_id"`
	Source         json.RawMessage `json:"source"` // "cli", or an object for sub-agents
}

type subAgentSource struct {
	SubAgent struct {
		ThreadSpawn struct {
			ParentThreadID string `json:"parent_thread_id"`
		} `json:"thread_spawn"`
	} `json:"subagent"`
}

// SessionMeta parses a rollout's session_meta line. ok is false for any other
// line, and for a session_meta without an id.
func SessionMeta(line []byte) (Meta, bool) {
	var l rolloutLine
	if json.Unmarshal(line, &l) != nil || l.Type != "session_meta" {
		return Meta{}, false
	}
	var p metaPayload
	if json.Unmarshal(l.Payload, &p) != nil || p.ID == "" {
		return Meta{}, false
	}
	m := Meta{
		ID:             p.ID,
		Cwd:            p.Cwd,
		Originator:     p.Originator,
		ParentThreadID: p.ParentThreadID,
		ForkedFromID:   p.ForkedFromID,
	}
	if m.ParentThreadID == "" {
		var src subAgentSource
		if json.Unmarshal(p.Source, &src) == nil {
			m.ParentThreadID = src.SubAgent.ThreadSpawn.ParentThreadID
		}
	}
	for _, ts := range []string{p.Timestamp, l.Timestamp} {
		if t, err := time.Parse(time.RFC3339Nano, ts); err == nil {
			m.Timestamp = t
			break
		}
	}
	return m, true
}
