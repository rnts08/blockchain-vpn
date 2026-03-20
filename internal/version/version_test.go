package version

import (
	"strings"
	"testing"
)

func TestString(t *testing.T) {
	t.Parallel()

	versionStr := String()

	if !strings.Contains(versionStr, Version) {
		t.Errorf("String() should contain Version %q", Version)
	}

	if !strings.Contains(versionStr, GitCommit) {
		t.Errorf("String() should contain GitCommit %q", GitCommit)
	}

	if !strings.Contains(versionStr, BuildDate) {
		t.Errorf("String() should contain BuildDate %q", BuildDate)
	}

	if !strings.HasPrefix(versionStr, "bcvpn") {
		t.Errorf("String() should start with \"bcvpn\", got %q", versionStr)
	}
}

func TestVersionFormat(t *testing.T) {
	t.Parallel()

	if Version == "" {
		t.Error("Version should not be empty")
	}

	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Errorf("Version should be in semver format (x.y.z), got %q", Version)
	}
}

func TestGitCommit(t *testing.T) {
	t.Parallel()

	if GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
}

func TestBuildDate(t *testing.T) {
	t.Parallel()

	if BuildDate == "" {
		t.Error("BuildDate should not be empty")
	}
}

func TestStringFormat(t *testing.T) {
	t.Parallel()

	versionStr := String()

	expectedParts := []string{"bcvpn", Version, "commit=", GitCommit, "built=", BuildDate}
	for _, part := range expectedParts {
		if !strings.Contains(versionStr, part) {
			t.Errorf("String() should contain %q, got %q", part, versionStr)
		}
	}
}

func BenchmarkString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = String()
	}
}
