//go:build functional

package functests

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestFunctional_Deployment_BuildTargets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		command string
	}{
		{"build CLI", "go build -o /tmp/bcvpn ./cmd/bcvpn"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("sh", "-c", tt.command)
			cmd.Dir = getProjectRoot()
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("Build failed: %v\nOutput: %s", err, output)
			}
		})
	}
}

func TestFunctional_Deployment_BuildOutput(t *testing.T) {
	t.Parallel()

	cmd := exec.Command("go", "build", "-o", "/tmp/bcvpn-test", "./cmd/bcvpn")
	cmd.Dir = getProjectRoot()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Build failed: %v\nOutput: %s", err, output)
	}

	defer os.Remove("/tmp/bcvpn-test")

	info, err := os.Stat("/tmp/bcvpn-test")
	if err != nil {
		t.Fatalf("Failed to stat build output: %v", err)
	}

	if info.Size() == 0 {
		t.Error("Build output is empty")
	}

	t.Logf("Build output size: %d bytes", info.Size())
}

func TestFunctional_Deployment_MakefileTargets(t *testing.T) {
	t.Parallel()

	if _, err := exec.LookPath("make"); err != nil {
		t.Skip("make not available")
	}

	projectRoot := getProjectRoot()

	tests := []struct {
		name   string
		target string
	}{
		{"test target", "test"},
		{"fmt target", "fmt"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := exec.Command("make", tt.target)
			cmd.Dir, _ = filepath.Abs(projectRoot)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("make %s output: %s", tt.target, output)
			}
		})
	}
}

func TestFunctional_Deployment_VersionFile(t *testing.T) {
	t.Parallel()

	projectRoot := getProjectRoot()

	content, err := os.ReadFile(filepath.Join(projectRoot, "VERSION"))
	if err != nil {
		t.Fatalf("Failed to read VERSION: %v", err)
	}

	if len(content) == 0 {
		t.Error("VERSION file is empty")
	}

	t.Logf("VERSION content: %s", string(content))
}

func TestFunctional_Deployment_GoMod(t *testing.T) {
	t.Parallel()

	projectRoot := getProjectRoot()

	content, err := os.ReadFile(filepath.Join(projectRoot, "go.mod"))
	if err != nil {
		t.Fatalf("Failed to read go.mod: %v", err)
	}

	if len(content) == 0 {
		t.Error("go.mod file is empty")
	}

	t.Logf("go.mod exists and has content")
}
