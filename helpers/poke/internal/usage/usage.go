// Package usage totals Claude Code and Codex token usage per local day for the
// last 14 days, reading the session logs incrementally.
//
// Three state files live in state.Dir():
//
//	usage.json          what the poke UI reads: 14 days, oldest first
//	usage-offsets.json  per log file: how far it has been read
//	usage-ledger.json   per message/response id: its day and token count
//
// The ledger is what makes every re-read harmless. A message that shows up
// again (streaming repeats, resumed sessions copying history, a log file
// rewritten shorter) is merged by id, never added twice.
package usage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"herdr-poke/internal/state"
)

// Days is how many days usage.json covers, today included.
const Days = 14

const (
	usageFile   = "usage.json"
	offsetsFile = "usage-offsets.json"
	ledgerFile  = "usage-ledger.json"
	lockFile    = "usage.lock"
)

// Roots are the log folders to read.
type Roots struct {
	ClaudeProjects string // ~/.claude/projects
	CodexSessions  string // ~/.codex/sessions
}

// DefaultRoots are the real log folders under $HOME.
func DefaultRoots() Roots {
	home, err := os.UserHomeDir()
	if err != nil {
		home = os.TempDir()
	}
	return Roots{
		ClaudeProjects: filepath.Join(home, ".claude", "projects"),
		CodexSessions:  filepath.Join(home, ".codex", "sessions"),
	}
}

// Day is one entry of usage.json.
type Day struct {
	Date   string `json:"date"` // YYYY-MM-DD
	Claude int    `json:"claude"`
	Codex  int    `json:"codex"`
}

// Output is the content of usage.json.
type Output struct {
	Days    []Day `json:"days"`
	Updated int64 `json:"updated"`
}

// progress is how far one log file has been read. Offset is the end of the
// last complete line; Size is the file size at that read, so a smaller size
// later means the file was replaced.
type progress struct {
	Size   int64 `json:"size"`
	Offset int64 `json:"offset"`
}

type entry struct {
	Day    string `json:"d"` // YYYY-MM-DD of the earliest sighting
	Tokens int    `json:"t"` // the largest count seen: a streamed message grows until complete
}

type ledger struct {
	Claude map[string]entry `json:"claude"`
	Codex  map[string]entry `json:"codex"`
}

// add merges one sighting into m and reports whether m changed.
func add(m map[string]entry, id, day string, tokens int) bool {
	old, seen := m[id]
	merged := entry{Day: day, Tokens: tokens}
	if seen {
		merged = old
		if day < merged.Day {
			merged.Day = day
		}
		if tokens > merged.Tokens {
			merged.Tokens = tokens
		}
		if merged == old {
			return false
		}
	}
	m[id] = merged
	return true
}

// Run brings usage.json up to date as of now. Days are local days in now's
// location. Only what the logs gained since the last run is read.
func Run(now time.Time, roots Roots) error {
	return state.WithLock(lockFile, func() error { return run(now, roots) })
}

func run(now time.Time, roots Roots) error {
	loc := now.Location()
	dayStart := func(back int) time.Time {
		return time.Date(now.Year(), now.Month(), now.Day()-back, 0, 0, 0, 0, loc)
	}
	dates := make([]string, Days) // oldest first
	for i := range dates {
		dates[i] = dayStart(Days - 1 - i).Format("2006-01-02")
	}
	oldest, today := dates[0], dates[Days-1]

	prev := map[string]progress{}
	_ = state.ReadJSON(offsetsFile, &prev)
	led := ledger{}
	if state.ReadJSON(ledgerFile, &led) != nil {
		prev = map[string]progress{} // offsets without their ledger would undercount: read everything again
		led = ledger{}
	}
	if led.Claude == nil {
		led.Claude = map[string]entry{}
	}
	if led.Codex == nil {
		led.Codex = map[string]entry{}
	}

	dirty := false
	for _, m := range []map[string]entry{led.Claude, led.Codex} {
		for id, e := range m {
			if e.Day < oldest {
				delete(m, id) // fell out of the window
				dirty = true
			}
		}
	}

	next := map[string]progress{}
	scan := func(root string, parse func([]byte) (record, bool), into map[string]entry) {
		for _, path := range logFiles(root, dayStart(Days-1)) {
			p, err := readNew(path, prev[path], func(line []byte) {
				r, ok := parse(line)
				if !ok {
					return
				}
				day := r.at.In(loc).Format("2006-01-02")
				if day < oldest || day > today {
					return // outside the window; a clock ahead of ours counts as nothing
				}
				if add(into, r.id, day, r.tokens) {
					dirty = true
				}
			})
			if err != nil {
				if old, ok := prev[path]; ok {
					next[path] = old
				}
				continue
			}
			next[path] = p
		}
	}
	scan(roots.ClaudeProjects, parseClaude, led.Claude)
	scan(roots.CodexSessions, parseCodex, led.Codex)

	// Ledger first: if we stop before the offsets are saved, the next run reads
	// the same lines again, and the ledger absorbs them.
	if dirty {
		if err := state.WriteJSON(ledgerFile, led); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(next, prev) {
		if err := state.WriteJSON(offsetsFile, next); err != nil {
			return err
		}
	}

	out := Output{Days: make([]Day, Days), Updated: now.Unix()}
	index := make(map[string]int, Days)
	for i, d := range dates {
		out.Days[i].Date = d
		index[d] = i
	}
	for _, e := range led.Claude {
		if i, ok := index[e.Day]; ok {
			out.Days[i].Claude += e.Tokens
		}
	}
	for _, e := range led.Codex {
		if i, ok := index[e.Day]; ok {
			out.Days[i].Codex += e.Tokens
		}
	}
	return state.WriteJSON(usageFile, out)
}

// logFiles lists the *.jsonl files under root (subagents/ included) modified
// at or after since. A log is append-only, so an older file holds nothing newer.
func logFiles(root string, since time.Time) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable entry (or no such root): skip it
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		if info, err := d.Info(); err == nil && info.Mode().IsRegular() && !info.ModTime().Before(since) {
			files = append(files, path)
		}
		return nil
	})
	return files
}

// readNew calls fn for every complete line of path after prev.Offset, and
// returns the new progress. A file smaller than before was replaced, so it is
// read from the start. A last line without its newline is passed to fn too (it
// may be complete), but not consumed: the next run sees it again.
func readNew(path string, prev progress, fn func(line []byte)) (progress, error) {
	f, err := os.Open(path)
	if err != nil {
		return progress{}, err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return progress{}, err
	}
	size := info.Size()
	offset := prev.Offset
	if size < prev.Size || offset > size {
		offset = 0
	}
	if size == offset {
		return progress{Size: size, Offset: offset}, nil
	}
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return progress{}, err
	}
	r := bufio.NewReaderSize(io.LimitReader(f, size-offset), 1<<20)
	for {
		line, err := r.ReadBytes('\n')
		if err != nil { // io.EOF: what is left (maybe nothing) has no newline yet
			if len(line) > 0 {
				fn(line)
			}
			break
		}
		offset += int64(len(line))
		fn(line)
	}
	return progress{Size: size, Offset: offset}, nil
}

// record is one usage event of a provider.
type record struct {
	id     string
	at     time.Time
	tokens int
}

var (
	markUsage     = []byte(`"usage"`)
	markAssistant = []byte(`"assistant"`)
	markRecord    = []byte(`"token_usage_record"`)
)

// parseClaude reads an assistant line: input + output + cache creation + cache
// read tokens of message.message.id. Everything else is not a record.
func parseClaude(line []byte) (record, bool) {
	if !bytes.Contains(line, markAssistant) || !bytes.Contains(line, markUsage) {
		return record{}, false // cheap test first: most lines are not assistant turns
	}
	var l struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		RequestID string `json:"requestId"`
		Message   struct {
			ID    string `json:"id"`
			Usage *struct {
				Input         int `json:"input_tokens"`
				Output        int `json:"output_tokens"`
				CacheCreation int `json:"cache_creation_input_tokens"`
				CacheRead     int `json:"cache_read_input_tokens"`
			} `json:"usage"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &l) != nil || l.Type != "assistant" || l.Message.Usage == nil {
		return record{}, false
	}
	id := l.Message.ID
	if id == "" {
		id = l.RequestID
	}
	at, err := time.Parse(time.RFC3339Nano, l.Timestamp)
	if id == "" || err != nil {
		return record{}, false
	}
	u := l.Message.Usage
	return record{id: id, at: at, tokens: u.Input + u.Output + u.CacheCreation + u.CacheRead}, true
}

// parseCodex reads a token_usage_record line: payload.usage.total_tokens of
// payload.response_id.
func parseCodex(line []byte) (record, bool) {
	if !bytes.Contains(line, markRecord) {
		return record{}, false
	}
	var l struct {
		Type      string `json:"type"`
		Timestamp string `json:"timestamp"`
		Payload   struct {
			ResponseID string `json:"response_id"`
			Usage      struct {
				TotalTokens int `json:"total_tokens"`
			} `json:"usage"`
		} `json:"payload"`
	}
	if json.Unmarshal(line, &l) != nil || l.Type != "token_usage_record" || l.Payload.ResponseID == "" {
		return record{}, false
	}
	at, err := time.Parse(time.RFC3339Nano, l.Timestamp)
	if err != nil {
		return record{}, false
	}
	return record{id: l.Payload.ResponseID, at: at, tokens: l.Payload.Usage.TotalTokens}, true
}
