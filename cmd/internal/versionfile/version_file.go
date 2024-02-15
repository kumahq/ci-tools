package versionfile

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"

	"github.com/kumahq/ci-tools/cmd/internal/github"
)

type VersionEntry struct {
	Edition       string `yaml:"edition"`
	Version       string `yaml:"version"`
	Release       string `yaml:"release"`
	Latest        bool   `yaml:"latest,omitempty"`
	ReleaseDate   string `yaml:"releaseDate,omitempty"`
	EndOfLifeDate string `yaml:"endOfLifeDate,omitempty"`
	Branch        string `yaml:"branch"`
	Label         string `yaml:"label,omitempty"`
}

func (v VersionEntry) Less(o VersionEntry) bool {
	vV := semver.MustParse(strings.ReplaceAll(v.Version, "x", "0"))
	vO := semver.MustParse(strings.ReplaceAll(o.Version, "x", "0"))
	return vV.LessThan(vO)
}

func BuildVersionEntry(edition string, releaseName string, lifetimeMonths int, releases []github.GQLRelease) (VersionEntry, error) {
	latest := false
	for _, r := range releases {
		latest = latest || r.IsLatest
	}
	sort.Slice(releases, func(i, j int) bool {
		iv, _ := strconv.Atoi(strings.Split(releases[i].Name, ".")[2])
		jv, _ := strconv.Atoi(strings.Split(releases[j].Name, ".")[2])
		return iv < jv
	})
	releaseDate, EOLDate, err := extractStartAndEOLDates(lifetimeMonths, releases)
	if err != nil {
		return VersionEntry{}, err
	}
	// Retrieve the latest release that is not a draft.
	latestRelease := releases[len(releases)-1]
	for i := range releases {
		if releases[i].IsReleased() {
			latestRelease = releases[i]
		}
	}
	return VersionEntry{
		Release:       releaseName,
		Edition:       edition,
		Version:       latestRelease.Name,
		Latest:        latest,
		ReleaseDate:   releaseDate,
		EndOfLifeDate: EOLDate,
		Branch:        releases[0].Branch(),
	}, nil
}

func extractStartAndEOLDates(lifetimeMonths int, releases []github.GQLRelease) (string, string, error) {
	if !releases[0].IsReleased() {
		return "", "", nil
	}
	releaseDate, err := releases[0].ExtractReleaseDate()
	if err != nil {
		return "", "", fmt.Errorf("failed to extract release date for %s because of: %s", releases[0].Name, err.Error())
	}
	EOLDate := releaseDate.AddDate(0, lifetimeMonths, 0)
	return releaseDate.Format(time.DateOnly), EOLDate.Format(time.DateOnly), nil
}
