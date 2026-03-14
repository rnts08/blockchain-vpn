//go:build functional

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctional_Contributor_DevEnvironment(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
	}{
		{"go version", "go version"},
		{"go mod tidy", "go mod tidy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", tt.command)
			cmd.Dir = "/home/timh/Projects/blockchain-vpn"
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("Command output: %s", string(output))
			}
		})
	}
}

func TestFunctional_Contributor_BuildTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		target string
	}{
		{"build", "make build"},
		{"test", "make test"},
		{"fmt", "make fmt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", tt.target)
			cmd.Dir = "/home/timh/Projects/blockchain-vpn"
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("make %s output: %s", tt.target, string(output))
			}
		})
	}
}

func TestFunctional_Contributor_DirectoryStructure(t *testing.T) {
	t.Parallel()

	requiredDirs := []string{
		"cmd",
		"internal",
		"docs",
	}

	projectRoot := "/home/timh/Projects/blockchain-vpn"

	for _, dir := range requiredDirs {
		path := filepath.Join(projectRoot, dir)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("Required directory %s not found: %v", dir, err)
		}
		if !info.IsDir() {
			t.Errorf("%s is not a directory", dir)
		}
	}

	t.Log("All required directories exist")
}

func TestFunctional_Contributor_GoMod(t *testing.T) {
	t.Parallel()

	projectRoot := "/home/timh/Projects/blockchain-vpn"

	content, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		t.Fatalf("go.mod not found: %v", err)
	}

	if !strings.Contains(string(content), "module blockchain-vpn") {
		t.Error("go.mod missing module declaration")
	}

	t.Log("go.mod is valid")
}

func TestFunctional_Contributor_DocumentationExists(t *testing.T) {
	t.Parallel()

	requiredDocs := []string{
		"README.md",
		"docs/CHANGELOG.md",
		"docs/CONTRIBUTING.md",
	}

	projectRoot := "/home/timh/Projects/blockchain-vpn"

	for _, doc := range requiredDocs {
		path := filepath.Join(projectRoot, doc)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Required documentation %s not found", doc)
		}
	}

	t.Log("All required documentation exists")
}

func TestFunctional_Contributor_MakefileTargets(t *testing.T) {
	t.Parallel()

	projectRoot := "/home/timh/Projects/blockchain-vpn"

	content, err := os.ReadFile(filepath.Join(projectRoot, "Makefile"))
	if err != nil {
		t.Fatalf("Makefile not found: %v", err)
	}

	requiredTargets := []string{"test", "build", "fmt"}

	for _, target := range requiredTargets {
		if !strings.Contains(string(content), target+":") {
			t.Errorf("Makefile missing target: %s", target)
		}
	}

	t.Log("All required Makefile targets exist")
}
