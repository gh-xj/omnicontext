package adapters

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func Parse(kind, p string) ([]Session, error) {
	if kind != "claude" && kind != "codex" {
		return nil, fmt.Errorf("unsupported adapter: %s", kind)
	}
	files, err := collectJSONLFiles(kind, p)
	if err != nil {
		return nil, err
	}
	out := make([]Session, 0, len(files))
	for _, f := range files {
		var s Session
		if kind == "claude" {
			s, err = parseClaudeFile(f)
		} else {
			s, err = parseCodexFile(f)
		}
		if err != nil {
			continue
		}
		if s.SessionID == "" && len(s.Turns) == 0 {
			continue
		}
		if s.SessionID == "" {
			s.SessionID = strings.TrimSuffix(filepath.Base(f), filepath.Ext(f))
		}
		s.SessionType = kind
		s.SessionPath = f
		if s.SessionTitle == "" {
			s.SessionTitle = "imported " + kind + " session"
		}
		if s.Metadata == "" {
			s.Metadata = "{}"
		}
		out = append(out, s)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no parsable %s sessions found at %s", kind, p)
	}
	return out, nil
}

func collectJSONLFiles(kind, p string) ([]string, error) {
	st, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !st.IsDir() {
		if strings.HasSuffix(strings.ToLower(p), ".jsonl") {
			return []string{p}, nil
		}
		return nil, fmt.Errorf("file is not .jsonl: %s", p)
	}
	var out []string
	err = filepath.WalkDir(p, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if kind == "claude" && strings.Contains(path, string(filepath.Separator)+"subagents") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(path), ".jsonl") {
			out = append(out, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

type lineEnvelope struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	SessionID string          `json:"sessionId"`
	CWD       string          `json:"cwd"`
	Message   json.RawMessage `json:"message"`
	Payload   json.RawMessage `json:"payload"`
}

type messageRole struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type codexPayload struct {
	Type    string      `json:"type"`
	Role    string      `json:"role"`
	ID      string      `json:"id"`
	CWD     string      `json:"cwd"`
	Message interface{} `json:"message"`
	Content interface{} `json:"content"`
}

func parseClaudeFile(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	s := Session{}
	var started, last string
	var pendingUser string
	turnNo := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		var env lineEnvelope
		if err := json.Unmarshal(line, &env); err != nil {
			continue
		}
		if env.SessionID != "" && s.SessionID == "" {
			s.SessionID = env.SessionID
		}
		if env.CWD != "" && s.WorkspacePath == "" {
			s.WorkspacePath = env.CWD
		}
		if started == "" && env.Timestamp != "" {
			started = env.Timestamp
		}
		if env.Timestamp != "" {
			last = env.Timestamp
		}
		role := env.Type
		text := ""
		if len(env.Message) > 0 {
			var m messageRole
			if err := json.Unmarshal(env.Message, &m); err == nil {
				if m.Role != "" {
					role = m.Role
				}
				text = flattenContent(m.Content)
			}
		}
		if role == "user" {
			if pendingUser != "" {
				turnNo++
				s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, Timestamp: env.Timestamp})
			}
			pendingUser = strings.TrimSpace(text)
		} else if role == "assistant" {
			a := strings.TrimSpace(text)
			if pendingUser != "" || a != "" {
				turnNo++
				s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, AssistantSummary: a, Timestamp: env.Timestamp})
				pendingUser = ""
			}
		}
	}
	if err := sc.Err(); err != nil {
		return Session{}, err
	}
	if pendingUser != "" {
		s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, Timestamp: last})
	}
	s.StartedAt = started
	s.LastActivityAt = last
	s.SessionSummary = summaryFromTurns(s.Turns)
	if s.WorkspacePath == "" {
		s.WorkspacePath = filepath.Dir(path)
	}
	return s, nil
}

func parseCodexFile(path string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	s := Session{}
	var started, last string
	var pendingUser string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	for sc.Scan() {
		var env lineEnvelope
		if err := json.Unmarshal(sc.Bytes(), &env); err != nil {
			continue
		}
		if started == "" && env.Timestamp != "" {
			started = env.Timestamp
		}
		if env.Timestamp != "" {
			last = env.Timestamp
		}
		if env.Type == "session_meta" && len(env.Payload) > 0 {
			var p struct {
				ID  string `json:"id"`
				CWD string `json:"cwd"`
			}
			_ = json.Unmarshal(env.Payload, &p)
			if p.ID != "" {
				s.SessionID = p.ID
			}
			if p.CWD != "" && s.WorkspacePath == "" {
				s.WorkspacePath = p.CWD
			}
		}
		if env.Type == "event_msg" && len(env.Payload) > 0 {
			var ev struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(env.Payload, &ev)
			if ev.Type == "user_message" && strings.TrimSpace(ev.Message) != "" {
				if pendingUser != "" {
					s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, Timestamp: env.Timestamp})
				}
				pendingUser = strings.TrimSpace(ev.Message)
			}
			continue
		}
		if env.Type != "response_item" || len(env.Payload) == 0 {
			continue
		}
		var p codexPayload
		if err := json.Unmarshal(env.Payload, &p); err != nil {
			continue
		}
		if p.Type != "message" {
			continue
		}
		if p.ID != "" && s.SessionID == "" {
			s.SessionID = p.ID
		}
		if p.CWD != "" && s.WorkspacePath == "" {
			s.WorkspacePath = p.CWD
		}
		text := flattenContent(p.Content)
		if strings.TrimSpace(text) == "" {
			text = flattenContent(p.Message)
		}
		if p.Role == "user" {
			if pendingUser != "" {
				s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, Timestamp: env.Timestamp})
			}
			pendingUser = strings.TrimSpace(text)
		} else if p.Role == "assistant" {
			a := strings.TrimSpace(text)
			if pendingUser != "" || a != "" {
				s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, AssistantSummary: a, Timestamp: env.Timestamp})
				pendingUser = ""
			}
		}
	}
	if err := sc.Err(); err != nil {
		return Session{}, err
	}
	if pendingUser != "" {
		s.Turns = append(s.Turns, Turn{UserMessage: pendingUser, Timestamp: last})
	}
	s.StartedAt = started
	s.LastActivityAt = last
	s.SessionSummary = summaryFromTurns(s.Turns)
	if s.WorkspacePath == "" {
		s.WorkspacePath = filepath.Dir(path)
	}
	return s, nil
}

func flattenContent(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case []interface{}:
		parts := make([]string, 0, len(t))
		for _, it := range t {
			if m, ok := it.(map[string]interface{}); ok {
				if x, ok := m["text"].(string); ok && x != "" {
					parts = append(parts, x)
					continue
				}
				if x, ok := m["input_text"].(string); ok && x != "" {
					parts = append(parts, x)
					continue
				}
				if x, ok := m["output_text"].(string); ok && x != "" {
					parts = append(parts, x)
					continue
				}
				if x, ok := m["content"].(string); ok && x != "" {
					parts = append(parts, x)
					continue
				}
			}
		}
		return strings.Join(parts, "\n")
	case map[string]interface{}:
		if x, ok := t["text"].(string); ok && x != "" {
			return x
		}
		if x, ok := t["input_text"].(string); ok && x != "" {
			return x
		}
		if x, ok := t["output_text"].(string); ok && x != "" {
			return x
		}
		if x, ok := t["content"].(string); ok && x != "" {
			return x
		}
		b, _ := json.Marshal(t)
		return string(b)
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

func summaryFromTurns(turns []Turn) string {
	for _, t := range turns {
		candidate := strings.TrimSpace(t.AssistantSummary)
		if candidate != "" {
			if len(candidate) > 180 {
				return candidate[:180]
			}
			return candidate
		}
	}
	for _, t := range turns {
		candidate := strings.TrimSpace(t.UserMessage)
		if candidate != "" {
			if len(candidate) > 180 {
				return candidate[:180]
			}
			return candidate
		}
	}
	return ""
}
