package obs

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"debug", levelDebug},
		{"DEBUG", levelDebug},
		{"info", levelInfo},
		{"INFO", levelInfo},
		{"warn", levelWarn},
		{"WARN", levelWarn},
		{"error", levelError},
		{"ERROR", levelError},
		{"", levelInfo},
		{"unknown", levelInfo},
		{"warning", levelInfo}, // "warning" is not explicitly handled, falls through to default
	}

	for _, tt := range tests {
		result := parseLevel(tt.input)
		if result != tt.expected {
			t.Errorf("parseLevel(%q) = %d, want %d", tt.input, result, tt.expected)
		}
	}
}

func TestClassifyLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"fatal error occurred", "error"},
		{"Error: connection failed", "error"},
		{"something failed", "error"},
		{"some error happened", "error"},
		{"WARNING: disk full", "warn"},
		{"warning: low memory", "warn"},
		{"debug info here", "debug"},
		{"just a regular message", "info"},
		{"INFO: system started", "info"},
	}

	for _, tt := range tests {
		result, _ := classifyLevel(tt.input)
		if result != tt.expected {
			t.Errorf("classifyLevel(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestJsonLogWriterFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := &jsonLogWriter{
		component: "test",
		out:       buf,
		minLevel:  levelInfo,
	}

	writer.Write([]byte("debug message"))
	writer.Write([]byte("info message"))
	writer.Write([]byte("error message"))

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 lines (debug filtered), got %d", len(lines))
	}
}

func TestTextLogWriterFiltering(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := &textLogWriter{
		out:      buf,
		minLevel: levelWarn,
	}

	writer.Write([]byte("debug message"))
	writer.Write([]byte("warning: something"))
	writer.Write([]byte("error occurred"))

	output := buf.String()
	if output == "" {
		t.Error("expected some output")
	}
}

func TestConfigureLoggingJSON(t *testing.T) {
	ConfigureLogging("json", "debug", "test")
}

func TestConfigureLoggingText(t *testing.T) {
	ConfigureLogging("text", "info", "test")
}

func TestJsonLogWriterInvalid(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := &jsonLogWriter{
		component: "test",
		out:       buf,
		minLevel:  levelInfo,
	}

	n, err := writer.Write([]byte(""))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
}

func TestTextLogWriterInvalid(t *testing.T) {
	buf := &bytes.Buffer{}
	writer := &textLogWriter{
		out:      buf,
		minLevel: levelInfo,
	}

	n, err := writer.Write([]byte(""))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 bytes written, got %d", n)
	}
}
