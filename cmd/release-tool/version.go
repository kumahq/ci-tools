package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// NormalizeVersionTag ensures the version tag has the correct format (with or without v prefix)
// based on kumahq/kuma tagging conventions. Logs a warning if v prefix was auto-added.
//
// Tagging convention:
//   - >= 2.13.x or >= 3.x: always v prefix
//   - 2.12.x: v prefix if patch >= 4
//   - 2.11.x: v prefix if patch >= 8
//   - 2.10.x: v prefix if patch >= 9
//   - 2.7.x: v prefix if patch >= 20
//   - Other older versions: no v prefix
func NormalizeVersionTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return tag
	}

	v, err := semver.NewVersion(tag)
	if err != nil {
		return tag
	}

	if needsVPrefix(v) {
		_, _ = fmt.Fprintf(os.Stderr, "Warning: auto-adding 'v' prefix to tag %s -> v%s (kumahq/kuma uses v-prefixed tags for this version)\n", tag, tag)

		return "v" + tag
	}

	return tag
}

// needsVPrefix determines if a version should have a v prefix based on kumahq/kuma conventions.
func needsVPrefix(v *semver.Version) bool {
	major := v.Major()
	minor := v.Minor()
	patch := v.Patch()

	// Any version >= 3.x always needs v prefix
	if major >= 3 {
		return true
	}

	// For major version 2
	if major == 2 {
		// >= 2.13.x: always v prefix (new minors)
		if minor >= 13 {
			return true
		}

		// 2.12.x: v prefix if patch >= 4
		if minor == 12 && patch >= 4 {
			return true
		}

		// 2.11.x: v prefix if patch >= 8
		if minor == 11 && patch >= 8 {
			return true
		}

		// 2.10.x: v prefix if patch >= 9
		if minor == 10 && patch >= 9 {
			return true
		}

		// 2.7.x: v prefix if patch >= 20
		if minor == 7 && patch >= 20 {
			return true
		}
	}

	return false
}
