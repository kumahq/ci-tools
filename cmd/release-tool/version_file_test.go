package main

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestNewestUnreleasedBranch(t *testing.T) {
	tests := []struct {
		name           string
		latestReleased string
		branches       []string
		expected       string
	}{
		{
			name:           "unreleased major branch wins over latest release",
			latestReleased: "2.14.5",
			branches:       []string{"master", "release-2.14", "release-3.0"},
			expected:       "3.0.0",
		},
		{
			name:           "unreleased minor branch follows the same rule",
			latestReleased: "2.14.5",
			branches:       []string{"master", "release-2.14", "release-2.15"},
			expected:       "2.15.0",
		},
		{
			name:           "branch of the latest release line is not unreleased",
			latestReleased: "2.14.5",
			branches:       []string{"master", "release-2.14"},
			expected:       "",
		},
		{
			name:           "newest of several unreleased branches wins",
			latestReleased: "2.14.5",
			branches:       []string{"release-2.15", "release-3.0"},
			expected:       "3.0.0",
		},
		{
			name:           "non release branches are ignored",
			latestReleased: "2.14.5",
			branches:       []string{"master", "main", "feature/foo", "release-3", "release-3.x", "release-"},
			expected:       "",
		},
		{
			name:           "no branches at all",
			latestReleased: "2.14.5",
			branches:       nil,
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			latest := semver.MustParse(tt.latestReleased)
			got := newestUnreleasedBranch(latest, tt.branches)
			if tt.expected == "" {
				if got != nil {
					t.Errorf("newestUnreleasedBranch() = %v, want nil", got)
				}
				return
			}
			if got == nil || got.String() != tt.expected {
				t.Errorf("newestUnreleasedBranch() = %v, want %s", got, tt.expected)
			}
		})
	}
}

func TestBranchPatternAndName(t *testing.T) {
	v := semver.MustParse("3.0.0")
	if got := branchPattern(v); got != "3.0.x" {
		t.Errorf("branchPattern() = %s, want 3.0.x", got)
	}
	if got := branchName(v); got != "release-3.0" {
		t.Errorf("branchName() = %s, want release-3.0", got)
	}
}

func TestInsertBeforeDefault(t *testing.T) {
	tests := []struct {
		name       string
		branches   []string
		unreleased string
		expected   []string
	}{
		{
			name:       "inserted before master",
			branches:   []string{"release-2.13", "release-2.14", "master"},
			unreleased: "release-3.0",
			expected:   []string{"release-2.13", "release-2.14", "release-3.0", "master"},
		},
		{
			name:       "appended when master missing",
			branches:   []string{"release-2.14"},
			unreleased: "release-3.0",
			expected:   []string{"release-2.14", "release-3.0"},
		},
		{
			name:       "empty list",
			branches:   nil,
			unreleased: "release-3.0",
			expected:   []string{"release-3.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := insertBeforeDefault(tt.branches, tt.unreleased)
			if len(got) != len(tt.expected) {
				t.Fatalf("insertBeforeDefault() = %v, want %v", got, tt.expected)
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("insertBeforeDefault() = %v, want %v", got, tt.expected)
					break
				}
			}
		})
	}
}
