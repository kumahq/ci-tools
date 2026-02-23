package versionfile_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/kumahq/ci-tools/cmd/internal/github"
	"github.com/kumahq/ci-tools/cmd/internal/versionfile"
)

func TestBuildVersionEntry(t *testing.T) {
	type entry struct {
		desc                string
		inEdition           string
		inReleaseName       string
		inLifetimeMonths    int
		inLtsLifetimeMonths int
		inReleases          []github.GQLRelease
		out                 versionfile.VersionEntry
	}
	d1 := time.Date(2020, 12, 12, 2, 2, 2, 0, time.UTC)
	simpleCase := func(desc string, inReleases []github.GQLRelease, out versionfile.VersionEntry) entry {
		return entry{
			desc:                desc,
			inEdition:           "mesh",
			inReleaseName:       "1.2.x",
			inLifetimeMonths:    12,
			inLtsLifetimeMonths: 24,
			inReleases:          inReleases,
			out:                 out,
		}
	}
	for _, v := range []entry{
		simpleCase(
			"no draft",
			[]github.GQLRelease{
				{Name: "1.2.0", PublishedAt: d1},
				{Name: "1.2.1", PublishedAt: d1.Add(time.Hour * 24 * 8), IsLatest: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", Latest: true, ReleaseDate: "2020-12-12", EndOfLifeDate: "2021-12-12", Branch: "release-1.2"},
		),
		simpleCase(
			"draft at end",
			[]github.GQLRelease{
				{Name: "1.2.0", PublishedAt: d1},
				{Name: "1.2.1", PublishedAt: d1.Add(time.Hour * 24 * 8)},
				{Name: "1.2.2", IsDraft: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", ReleaseDate: "2020-12-12", EndOfLifeDate: "2021-12-12", Branch: "release-1.2"},
		),
		simpleCase(
			"never published uses the latest version",
			[]github.GQLRelease{
				{Name: "1.2.0", IsPrerelease: true},
				{Name: "1.2.1", IsPrerelease: true},
				{Name: "1.2.2", IsDraft: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.2", Release: "1.2.x", Branch: "release-1.2"},
		),
		simpleCase(
			"single release as draft",
			[]github.GQLRelease{
				{Name: "1.2.0", IsDraft: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.0", Release: "1.2.x", Branch: "release-1.2"},
		),
		simpleCase(
			"use date from description",
			[]github.GQLRelease{
				{Name: "1.2.0", Description: "> Released on 2019/01/01"},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.0", Release: "1.2.x", Latest: false, ReleaseDate: "2019-01-01", EndOfLifeDate: "2020-01-01", Branch: "release-1.2"},
		),
		simpleCase(
			"use lts from description",
			[]github.GQLRelease{
				{Name: "1.2.0", Description: "> LTS", PublishedAt: d1},
				{Name: "1.2.1", Description: "foo", PublishedAt: d1.Add(time.Hour * 48)},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", LTS: true, ReleaseDate: "2020-12-12", EndOfLifeDate: "2022-12-12", Branch: "release-1.2"},
		),
		simpleCase(
			"ignore lts from description on not the first release",
			[]github.GQLRelease{
				{Name: "1.2.1", Description: "> LTS", PublishedAt: d1.Add(time.Hour * 48)},
				{Name: "1.2.0", Description: "foo", PublishedAt: d1},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", ReleaseDate: "2020-12-12", EndOfLifeDate: "2021-12-12", Branch: "release-1.2"},
		),
		simpleCase(
			"strips v-prefix from release names",
			[]github.GQLRelease{
				{Name: "v1.2.0", PublishedAt: d1},
				{Name: "v1.2.1", PublishedAt: d1.Add(time.Hour * 24 * 8), IsLatest: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", Latest: true, ReleaseDate: "2020-12-12", EndOfLifeDate: "2021-12-12", Branch: "release-1.2"},
		),
		simpleCase(
			"extended adds months on top of regular lifetime",
			[]github.GQLRelease{
				{Name: "1.2.0", Description: "> ExtensionMonths: 6", PublishedAt: d1},
				{Name: "1.2.1", PublishedAt: d1.Add(time.Hour * 24 * 8), IsLatest: true},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", Latest: true, ReleaseDate: "2020-12-12", EndOfLifeDate: "2022-06-12", Branch: "release-1.2", ExtensionMonths: 6},
		),
		simpleCase(
			"extended combined with lts adds months on top of lts lifetime",
			[]github.GQLRelease{
				{Name: "1.2.0", Description: "> LTS\n> ExtensionMonths: 6", PublishedAt: d1},
				{Name: "1.2.1", PublishedAt: d1.Add(time.Hour * 48)},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", LTS: true, ReleaseDate: "2020-12-12", EndOfLifeDate: "2023-06-12", Branch: "release-1.2", ExtensionMonths: 6},
		),
		simpleCase(
			"extended combined with custom release date",
			[]github.GQLRelease{
				{Name: "1.2.0", Description: "> Released on 2019/01/01\n> ExtensionMonths: 6"},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.0", Release: "1.2.x", ReleaseDate: "2019-01-01", EndOfLifeDate: "2020-07-01", Branch: "release-1.2", ExtensionMonths: 6},
		),
		simpleCase(
			"extended on non-first release is ignored",
			[]github.GQLRelease{
				{Name: "1.2.1", Description: "> ExtensionMonths: 6", PublishedAt: d1.Add(time.Hour * 48)},
				{Name: "1.2.0", Description: "foo", PublishedAt: d1},
			},
			versionfile.VersionEntry{Edition: "mesh", Version: "1.2.1", Release: "1.2.x", ReleaseDate: "2020-12-12", EndOfLifeDate: "2021-12-12", Branch: "release-1.2"},
		),
	} {
		t.Run(v.desc, func(t *testing.T) {
			res, err := versionfile.BuildVersionEntry(v.inEdition, v.inReleaseName, v.inLifetimeMonths, v.inLtsLifetimeMonths, v.inReleases)
			if err != nil {
				t.Errorf("%+v", err)
			}
			if !reflect.DeepEqual(res, v.out) {
				t.Errorf("not the same item,\n got:\n%#v\nexpected:\n%#v", res, v.out)
			}
		})
	}
}
