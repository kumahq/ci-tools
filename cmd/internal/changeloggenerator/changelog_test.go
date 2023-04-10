package changeloggenerator_test

import (
	"reflect"
	"testing"

	"github.com/kumahq/ci-tools/cmd/internal/changeloggenerator"
)

func TestChangelogGenerator(t *testing.T) {
	type entry struct {
		desc string
		in   []changeloggenerator.CommitInfo
		out  changeloggenerator.Changelog
	}
	for _, v := range []entry{
		{
			"2 simple entries",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.5", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): bump foo from 1.2.5 to 1.2.7", PrNumber: 124, Author: "a"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.7", Authors: []string{"@a"}, PullRequests: []int{123, 124}},
			},
		},
		{
			"with bump",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.5", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from 1.2.5 to 1.2.7", PrNumber: 124, Author: "a"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.7", Authors: []string{"@a"}, PullRequests: []int{123, 124}},
			},
		},
		{
			"forward then rollback",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.8", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from 1.2.8 to 1.2.3", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.3", Authors: []string{"@a", "@b"}, PullRequests: []int{123, 124}},
			},
		},
		{
			"not semver",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from deadbeef to deadbabe", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from foew to dead", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from deadbeef to dead", Authors: []string{"@a", "@b"}, PullRequests: []int{123, 124}},
			},
		},
		{
			"different deps don't get collapsed",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from deadbeef to deadbabe", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump bar from foew to dead", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump bar from foew to dead", Authors: []string{"@b"}, PullRequests: []int{124}},
				{Desc: "chore(deps): bump foo from deadbeef to deadbabe", Authors: []string{"@a"}, PullRequests: []int{123}},
			},
		},
	} {
		t.Run(v.desc, func(t *testing.T) {
			res, err := changeloggenerator.New("kumahq/kuma", v.in)
			if err != nil {
				t.Errorf("%+v", err)
			}
			if len(res) != len(v.out) {
				t.Errorf("not the same length got: %+v expected: %+v", res, v.out)
			}
			for i := range v.out {
				v.out[i].Repo = "kumahq/kuma"
				if !reflect.DeepEqual(res[i], v.out[i]) {
					t.Errorf("not the same item at idx %d: %v expected: %v", i, res[i], v.out[i])
				}
			}
		})
	}
}
