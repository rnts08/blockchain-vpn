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

type jsonLogWriter struct {
	component string
	out       io.Writer
	mu        sync.Mutex
}

func ConfigureLogging(format, component string) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "json":
		log.SetFlags(0)
		log.SetOutput(&jsonLogWriter{
			component: component,
			out:       os.Stdout,
		})
	default:
		log.SetFlags(log.LstdFlags)
		log.SetOutput(os.Stdout)
	}
}

func (w *jsonLogWriter) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	entry := map[string]any{
		"ts":        time.Now().UTC().Format(time.RFC3339Nano),
		"level":     "info",
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
