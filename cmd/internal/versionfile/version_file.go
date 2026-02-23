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
	Edition        string `yaml:"edition"`
	Version        string `yaml:"version"`
	Release        string `yaml:"release"`
	Latest         bool   `yaml:"latest,omitempty"`
	ReleaseDate    string `yaml:"releaseDate,omitempty"`
	EndOfLifeDate  string `yaml:"endOfLifeDate,omitempty"`
	Branch         string `yaml:"branch"`
	Label          string `yaml:"label,omitempty"`
	LTS            bool   `yaml:"lts,omitempty"`
	ExtendedMonths int    `yaml:"extendedMonths,omitempty"`
}

func (v VersionEntry) Less(o VersionEntry) bool {
	vV := semver.MustParse(strings.ReplaceAll(v.Version, "x", "0"))
	vO := semver.MustParse(strings.ReplaceAll(o.Version, "x", "0"))
	return vV.LessThan(vO)
}

func BuildVersionEntry(edition string, releaseName string, lifetimeMonths int, ltslifetimeMonths int, releases []github.GQLRelease) (VersionEntry, error) {
	out := VersionEntry{
		Release: releaseName,
		Edition: edition,
		Branch:  releases[0].Branch(),
	}
	for _, r := range releases {
		out.Latest = out.Latest || r.IsLatest
	}
	sort.Slice(releases, func(i, j int) bool {
		iv, _ := strconv.Atoi(strings.Split(releases[i].Name, ".")[2])
		jv, _ := strconv.Atoi(strings.Split(releases[j].Name, ".")[2])
		return iv < jv
	})
	if releases[0].IsReleased() {
		lifetime := lifetimeMonths
		if releases[0].IsLTS() {
			out.LTS = true
			lifetime = ltslifetimeMonths
		}
		if ext := releases[0].ExtendedMonths(); ext > 0 {
			lifetime += ext
			out.ExtendedMonths = ext
		}
		releaseDate, err := releases[0].ExtractReleaseDate()
		if err != nil {
			return out, fmt.Errorf("failed to extract release date for %s because of: %s", releases[0].Name, err.Error())
		}
		EOLDate := releaseDate.AddDate(0, lifetime, 0)
		out.ReleaseDate = releaseDate.Format(time.DateOnly)
		out.EndOfLifeDate = EOLDate.Format(time.DateOnly)
	}
	// Retrieve the latest release that is not a draft.
	latestRelease := releases[len(releases)-1]
	for i := range releases {
		if releases[i].IsReleased() {
			latestRelease = releases[i]
		}
	}
	out.Version = strings.TrimPrefix(latestRelease.Name, "v")
	return out, nil
}
