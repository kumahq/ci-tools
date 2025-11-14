package main

import (
	"strings"
	"testing"
)

func TestVersionPrefixStripping(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "version with v prefix",
			input:    "v2.11.8",
			expected: "2.11.8",
		},
		{
			name:     "version without v prefix",
			input:    "2.11.8",
			expected: "2.11.8",
		},
		{
			name:     "version with v prefix major only",
			input:    "v2",
			expected: "2",
		},
		{
			name:     "version with v prefix and prerelease",
			input:    "v2.11.8-preview.v12fbf5f56",
			expected: "2.11.8-preview.v12fbf5f56",
		},
		{
			name:     "version without v prefix and prerelease",
			input:    "2.11.8-preview.v12fbf5f56",
			expected: "2.11.8-preview.v12fbf5f56",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "just v",
			input:    "v",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.TrimPrefix(tt.input, "v")
			if result != tt.expected {
				t.Errorf("TrimPrefix(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHelmChartExpectedName(t *testing.T) {
	tests := []struct {
		name         string
		repo         string
		release      string
		expectedName string
	}{
		{
			name:         "kuma with v prefix",
			repo:         "kumahq/kuma",
			release:      "v2.11.8",
			expectedName: "kuma-2.11.8",
		},
		{
			name:         "kuma without v prefix",
			repo:         "kumahq/kuma",
			release:      "2.11.8",
			expectedName: "kuma-2.11.8",
		},
		{
			name:         "kong-mesh with v prefix",
			repo:         "Kong/kong-mesh",
			release:      "v2.11.8",
			expectedName: "kong-mesh-2.11.8",
		},
		{
			name:         "kong-mesh without v prefix",
			repo:         "Kong/kong-mesh",
			release:      "2.11.8",
			expectedName: "kong-mesh-2.11.8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseVersion := strings.TrimPrefix(tt.release, "v")
			repoName := strings.Split(tt.repo, "/")[1]
			actualName := repoName + "-" + releaseVersion
			if actualName != tt.expectedName {
				t.Errorf("Expected helm chart name %q, got %q", tt.expectedName, actualName)
			}
		})
	}
}

func TestDockerImageExpectedTag(t *testing.T) {
	tests := []struct {
		name        string
		dockerRepo  string
		imageName   string
		release     string
		expectedTag string
	}{
		{
			name:        "kuma-cp with v prefix",
			dockerRepo:  "kumahq",
			imageName:   "kuma-cp",
			release:     "v2.11.8",
			expectedTag: "kumahq/kuma-cp:2.11.8",
		},
		{
			name:        "kuma-cp without v prefix",
			dockerRepo:  "kumahq",
			imageName:   "kuma-cp",
			release:     "2.11.8",
			expectedTag: "kumahq/kuma-cp:2.11.8",
		},
		{
			name:        "kumactl with v prefix",
			dockerRepo:  "kumahq",
			imageName:   "kumactl",
			release:     "v2.12.4",
			expectedTag: "kumahq/kumactl:2.12.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseVersion := strings.TrimPrefix(tt.release, "v")
			actualTag := tt.dockerRepo + "/" + tt.imageName + ":" + releaseVersion
			if actualTag != tt.expectedTag {
				t.Errorf("Expected docker tag %q, got %q", tt.expectedTag, actualTag)
			}
		})
	}
}

func TestBinaryURLExpectedVersion(t *testing.T) {
	tests := []struct {
		name            string
		release         string
		binary          string
		expectedVersion string
	}{
		{
			name:            "darwin-amd64 with v prefix",
			release:         "v2.11.8",
			binary:          "darwin-amd64",
			expectedVersion: "2.11.8",
		},
		{
			name:            "darwin-amd64 without v prefix",
			release:         "2.11.8",
			binary:          "darwin-amd64",
			expectedVersion: "2.11.8",
		},
		{
			name:            "linux-arm64 with v prefix",
			release:         "v2.12.4",
			binary:          "linux-arm64",
			expectedVersion: "2.12.4",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseVersion := strings.TrimPrefix(tt.release, "v")
			if releaseVersion != tt.expectedVersion {
				t.Errorf("Expected version %q, got %q", tt.expectedVersion, releaseVersion)
			}
		})
	}
}

func TestReleaseTagNormalization(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectedTag string
	}{
		{
			name:        "version without v prefix",
			input:       "2.11.8",
			expectedTag: "v2.11.8",
		},
		{
			name:        "version with v prefix",
			input:       "v2.11.8",
			expectedTag: "v2.11.8",
		},
		{
			name:        "version with v prefix and prerelease",
			input:       "v2.11.8-preview.v12fbf5f56",
			expectedTag: "v2.11.8-preview.v12fbf5f56",
		},
		{
			name:        "version without v prefix and prerelease",
			input:       "2.11.8-preview.v12fbf5f56",
			expectedTag: "v2.11.8-preview.v12fbf5f56",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			releaseTag := tt.input
			if !strings.HasPrefix(releaseTag, "v") {
				releaseTag = "v" + releaseTag
			}
			if releaseTag != tt.expectedTag {
				t.Errorf("Expected tag %q, got %q", tt.expectedTag, releaseTag)
			}
		})
	}
}
