package main

import (
	"testing"

	"github.com/Masterminds/semver/v3"
)

func TestNeedsVPrefix(t *testing.T) {
	tests := []struct {
		version  string
		expected bool
	}{
		// Major >= 3: always v prefix
		{"3.0.0", true},
		{"3.1.5", true},
		{"4.0.0", true},

		// 2.13.x and higher minors: always v prefix
		{"2.13.0", true},
		{"2.13.5", true},
		{"2.14.0", true},
		{"2.20.0", true},

		// 2.12.x: v prefix if patch >= 4
		{"2.12.0", false},
		{"2.12.3", false},
		{"2.12.4", true},
		{"2.12.5", true},
		{"2.12.10", true},

		// 2.11.x: v prefix if patch >= 8
		{"2.11.0", false},
		{"2.11.7", false},
		{"2.11.8", true},
		{"2.11.9", true},
		{"2.11.15", true},

		// 2.10.x: v prefix if patch >= 9
		{"2.10.0", false},
		{"2.10.8", false},
		{"2.10.9", true},
		{"2.10.10", true},
		{"2.10.20", true},

		// 2.7.x: v prefix if patch >= 20
		{"2.7.0", false},
		{"2.7.19", false},
		{"2.7.20", true},
		{"2.7.21", true},
		{"2.7.30", true},

		// Other 2.x versions: no v prefix
		{"2.9.0", false},
		{"2.9.10", false},
		{"2.8.0", false},
		{"2.8.8", false},
		{"2.6.0", false},
		{"2.6.15", false},
		{"2.5.0", false},
		{"2.4.0", false},

		// 1.x versions: no v prefix
		{"1.0.0", false},
		{"1.5.10", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			v := semver.MustParse(tt.version)
			got := needsVPrefix(v)
			if got != tt.expected {
				t.Errorf("needsVPrefix(%s) = %v, want %v", tt.version, got, tt.expected)
			}
		})
	}
}

func TestNormalizeVersionTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// Already has v prefix and needs it - unchanged
		{"v2.12.4", "v2.12.4"},
		{"v2.11.8", "v2.11.8"},
		{"v3.0.0", "v3.0.0"},

		// Needs v prefix - should be added
		{"2.12.4", "v2.12.4"},
		{"2.12.5", "v2.12.5"},
		{"2.11.8", "v2.11.8"},
		{"2.10.9", "v2.10.9"},
		{"2.7.20", "v2.7.20"},
		{"2.13.0", "v2.13.0"},
		{"3.0.0", "v3.0.0"},

		// Doesn't need v prefix - unchanged
		{"2.12.3", "2.12.3"},
		{"2.11.7", "2.11.7"},
		{"2.10.8", "2.10.8"},
		{"2.7.19", "2.7.19"},
		{"2.9.0", "2.9.0"},
		{"2.8.5", "2.8.5"},

		// Has v prefix but shouldn't - should be removed
		{"v2.12.3", "2.12.3"},
		{"v2.11.7", "2.11.7"},
		{"v2.10.8", "2.10.8"},
		{"v2.7.19", "2.7.19"},
		{"v2.9.0", "2.9.0"},
		{"v2.8.5", "2.8.5"},
		{"v1.5.0", "1.5.0"},

		// Invalid version - unchanged
		{"invalid", "invalid"},
		{"vinvalid", "vinvalid"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizeVersionTag(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizeVersionTag(%s) = %s, want %s", tt.input, got, tt.expected)
			}
		})
	}
}
