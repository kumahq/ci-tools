package main

import (
	"testing"

	"github.com/Masterminds/semver/v3"

	"github.com/kumahq/ci-tools/cmd/internal/github"
)

func TestLatestReleasedVersion(t *testing.T) {
	tests := []struct {
		name     string
		releases []github.GQLRelease
		expected string
	}{
		{
			name: "draft release does not mask its own branch",
			releases: []github.GQLRelease{
				{Name: "v2.14.5"},
				{Name: "v3.0.0", IsDraft: true},
			},
			expected: "2.14.5",
		},
		{
			name: "prereleases are ignored",
			releases: []github.GQLRelease{
				{Name: "v2.14.5"},
				{Name: "v3.0.0-rc1", IsPrerelease: true},
			},
			expected: "2.14.5",
		},
		{
			name: "highest published wins",
			releases: []github.GQLRelease{
				{Name: "v2.13.11"},
				{Name: "v2.14.5"},
				{Name: "v2.7.30"},
			},
			expected: "2.14.5",
		},
		{
			name:     "no published releases",
			releases: []github.GQLRelease{{Name: "v3.0.0", IsDraft: true}},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := latestReleasedVersion(tt.releases)
			if tt.expected == "" {
				if got != nil {
					t.Errorf("latestReleasedVersion() = %v, want nil", got)
				}
				return
			}
			if got == nil || got.String() != tt.expected {
				t.Errorf("latestReleasedVersion() = %v, want %s", got, tt.expected)
			}
		})
	}
}

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
			name:           "nil latest treats every release branch as unreleased",
			latestReleased: "",
			branches:       []string{"master", "release-3.0"},
			expected:       "3.0.0",
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
			var latest *semver.Version
			if tt.latestReleased != "" {
				latest = semver.MustParse(tt.latestReleased)
			}
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
