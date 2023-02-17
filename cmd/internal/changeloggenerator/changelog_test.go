package changeloggenerator_test

import (
	"github.com/kumahq/ci-tools/cmd/internal/changeloggenerator"
	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	"testing"
)

func TestRun(t *testing.T) {
	RegisterTestingT(t)
	RunSpecs(t, "changeloggenerator")
}

var _ = Describe("New", func() {
	testNewFn := func(in []changeloggenerator.CommitInfo, out changeloggenerator.Changelog) {
		res, err := changeloggenerator.New("kumahq/kuma", in)
		Expect(err).ToNot(HaveOccurred())
		Expect(res).To(HaveLen(len(out)))
		for i := range out {
			out[i].Repo = "kumahq/kuma"
			Expect(res[i]).To(Equal(out[i]))
		}
	}
	var _ = table.DescribeTable("dependabot merge", testNewFn,
		table.Entry("2 simple entries",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.5", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): bump foo from 1.2.5 to 1.2.7", PrNumber: 124, Author: "a"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.7", Authors: []string{"@a"}, PullRequests: []int{123, 124}},
			},
		),
		table.Entry("with Bump",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.5", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from 1.2.5 to 1.2.7", PrNumber: 124, Author: "a"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.7", Authors: []string{"@a"}, PullRequests: []int{123, 124}},
			},
		),
		table.Entry("forward then back",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from 1.2.4 to 1.2.8", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from 1.2.8 to 1.2.3", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from 1.2.4 to 1.2.3", Authors: []string{"@a", "@b"}, PullRequests: []int{123, 124}},
			},
		),
		table.Entry("not semver",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from deadbeef to deadbabe", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump foo from foew to dead", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump foo from deadbeef to dead", Authors: []string{"@a", "@b"}, PullRequests: []int{123, 124}},
			},
		),
		table.Entry("different deps don't get collapsed",
			[]changeloggenerator.CommitInfo{
				{PrTitle: "chore(deps): bump foo from deadbeef to deadbabe", PrNumber: 123, Author: "a"},
				{PrTitle: "chore(deps): Bump bar from foew to dead", PrNumber: 124, Author: "b"},
			},
			changeloggenerator.Changelog{
				{Desc: "chore(deps): bump bar from foew to dead", Authors: []string{"@b"}, PullRequests: []int{124}},
				{Desc: "chore(deps): bump foo from deadbeef to deadbabe", Authors: []string{"@a"}, PullRequests: []int{123}},
			},
		),
	)
})
