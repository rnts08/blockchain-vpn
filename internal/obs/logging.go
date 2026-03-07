package obs

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	levelDebug = iota
	levelInfo
	levelWarn
	levelError
)

type jsonLogWriter struct {
	component string
	out       io.Writer
	minLevel  int
	mu        sync.Mutex
}

type textLogWriter struct {
	out      io.Writer
	minLevel int
	mu       sync.Mutex
}

func ConfigureLogging(format, level, component string) {
	minLevel := parseLevel(level)
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		log.SetFlags(0)
		log.SetOutput(&jsonLogWriter{
			component: component,
			out:       os.Stdout,
			minLevel:  minLevel,
		})
	default:
		log.SetFlags(log.LstdFlags)
		log.SetOutput(&textLogWriter{
			out:      os.Stdout,
			minLevel: minLevel,
		})
	}
}

func (w *jsonLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	levelName, levelValue := classifyLevel(msg)
	if levelValue < w.minLevel {
		return len(p), nil
	}
	entry := map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"level":     levelName,
		"component": w.component,
		"msg":       msg,
	}
	b, marshalErr := json.Marshal(entry)
	if marshalErr != nil {
		return w.out.Write(p)
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.out.Write(append(b, '\n')); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *textLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	_, levelValue := classifyLevel(msg)
	if levelValue < w.minLevel {
		return len(p), nil
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.out.Write(p); err != nil {
		return 0, err
	}
	return len(p), nil
}

func parseLevel(v string) int {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return levelDebug
	case "warn":
		return levelWarn
	case "error":
		return levelError
	default:
		return levelInfo
	}
}

func classifyLevel(msg string) (string, int) {
	lower := strings.ToLower(strings.TrimSpace(msg))
	switch {
	case strings.Contains(lower, "fatal"):
		return "error", levelError
	case strings.HasPrefix(lower, "error"), strings.Contains(lower, " failed"), strings.Contains(lower, " error"):
		return "error", levelError
	case strings.HasPrefix(lower, "warning"), strings.Contains(lower, "warning"):
		return "warn", levelWarn
	case strings.HasPrefix(lower, "debug"):
		return "debug", levelDebug
	default:
		return "info", levelInfo
	}
}
