//go:build functional

package functests

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFunctional_Community_FeedbackChannels(t *testing.T) {
	t.Parallel()

	feedbackFiles := []string{
		"docs/CONTRIBUTING.md",
		"docs/COMMUNITY.md",
	}

	for _, file := range feedbackFiles {
		path := filepath.Join(getProjectRoot(), file)
		content, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("Feedback file %s not found: %v", file, err)
			continue
		}

		if len(content) < 100 {
			t.Errorf("Feedback file %s is too short", file)
		}

		t.Logf("Feedback file %s exists and has content", file)
	}
}

func TestFunctional_Community_DocumentationLinks(t *testing.T) {
	t.Parallel()

	readmePath := filepath.Join(getProjectRoot(), "README.md")
	content, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("README.md not found: %v", err)
	}

	requiredSections := []string{
		"version",
		"Support",
	}

	for _, section := range requiredSections {
		if !strings.Contains(strings.ToLower(string(content)), strings.ToLower(section)) {
			t.Errorf("README.md missing section: %s", section)
		}
	}

	t.Log("README.md has required sections")
}

func TestFunctional_Community_Changelog(t *testing.T) {
	t.Parallel()

	changelogPath := filepath.Join(getProjectRoot(), "docs/CHANGELOG.md")
	content, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("CHANGELOG.md not found: %v", err)
	}

	if !strings.Contains(string(content), "##") {
		t.Error("CHANGELOG.md missing version headers")
	}

	t.Log("CHANGELOG.md has version history")
}

func TestFunctional_Community_CodeOfConduct(t *testing.T) {
	t.Parallel()

	cocPaths := []string{
		"CODE_OF_CONDUCT.md",
		"docs/CODE_OF_CONDUCT.md",
	}

	found := false
	for _, path := range cocPaths {
		fullPath := filepath.Join(getProjectRoot(), path)
		if _, err := os.Stat(fullPath); err == nil {
			found = true
			break
		}
	}

	if !found {
		t.Log("Note: CODE_OF_CONDUCT.md not found (optional)")
	}

	t.Log("Code of conduct check complete")
}

func TestFunctional_Community_VersionTracking(t *testing.T) {
	t.Parallel()

	versionPath := filepath.Join(getProjectRoot(), "VERSION")
	content, err := os.ReadFile(versionPath)
	if err != nil {
		t.Fatalf("VERSION file not found: %v", err)
	}

	version := strings.TrimSpace(string(content))
	if version == "" {
		t.Error("VERSION file is empty")
	}

	t.Logf("VERSION is: %s", version)
}

func TestFunctional_Community_License(t *testing.T) {
	t.Parallel()

	licensePaths := []string{
		"LICENSE",
		"LICENSE.md",
	}

	found := false
	for _, path := range licensePaths {
		fullPath := filepath.Join(getProjectRoot(), path)
		if _, err := os.Stat(fullPath); err == nil {
			found = true
			break
		}
	}

	if !found {
		t.Log("Note: LICENSE file not found (optional)")
	}

	t.Log("License check complete")
}
