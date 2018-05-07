// release-note-gen generates release notes for a release from closed github PRs
// within the corresponding release milestone. For example, PRs that should be
// included in release note for 1.7.0 are all set to milestone "1.7 Release".
//
// For each closed PR, it generates one line release
// note in for form of:
//  description (#<pr>)
// For example:
//  Add grpc.Version string and use it in the UA (#1144)
//
// The PR's "Type" labels are used as the category of the change.
// For example, a PR with label `["Type: API change"]` will be classified as `"API change"`.
//
//  If a PR has more than one labels, the labels will be sorted in the order of
//  `"Dependencies", "API Change", "Feature", "Behavior Change", "Performance", "Bug", "Internal Cleanup", "Documentation", "Testing"`,
//  and the first one will be picked as the final category.
//
//  If a PR has no "Type" label (which shouldn't happen), it will be classified as
//  `"Bug"`.

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/github"
	"github.com/menghanl/release-note-gen/ghclient"
	"github.com/menghanl/release-note-gen/notes"
	"golang.org/x/oauth2"
)

var (
	token     = flag.String("token", "", "github token")
	release   = flag.String("release", "", "release number")
	owner     = flag.String("owner", "grpc", "github repo owner")
	repo      = flag.String("repo", "grpc-go", "github repo")
	thanks    = flag.Bool("thanks", false, "whether to include thank you note. grpc organization members are excluded")
	urwelcome = flag.String("urwelcome", "", "list of users to exclude from thank you note, format: user1,user2")
	verymuch  = flag.String("verymuch", "", "list of users to include in thank you note even if they are grpc org members, format: user1,user2")
)

const milestoneTitleSurfix = " Release" // For example, "1.7 Release".

func commaStringToSet(s string) map[string]struct{} {
	ret := make(map[string]struct{})
	tmp := strings.Split(s, ",")
	for _, t := range tmp {
		ret[t] = struct{}{}
	}
	return ret
}

func main() {
	flag.Parse()

	if *release == "" {
		fmt.Println("invalid release number, usage:")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Github API calls begin.

	var tc *http.Client
	if *token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	c := ghclient.New(tc, *owner, *repo)
	prs := c.GetMergedPRsForMilestone(*release + milestoneTitleSurfix)

	var (
		thanksFilter func(pr *github.Issue) bool
	)
	if *thanks {
		urwelcomeMap := commaStringToSet(*urwelcome)
		verymuchMap := commaStringToSet(*verymuch)
		grpcMembers := c.GetOrgMembers("grpc")

		thanksFilter = func(pr *github.Issue) bool {
			user := pr.GetUser().GetLogin()
			_, isGRPCMember := grpcMembers[user]
			_, isWelcome := urwelcomeMap[user]
			_, isVerymuch := verymuchMap[user]

			return *thanks && (isVerymuch || (!isGRPCMember && !isWelcome))
		}
	}

	// generate notes

	ns := notes.GenerateNotes(prs, notes.Filters{
		SpecialThanks: thanksFilter,
	})

	fmt.Printf("\n================ generated notes for org %q repo %q release %q ================\n\n", ns.Org, ns.Repo, ns.Version)
	for _, section := range ns.Sections {
		fmt.Printf("# %v\n\n", section.Name)
		for _, entry := range section.Entries {
			fmt.Printf(" * %v (#%v)\n", entry.Title, entry.IssueNumber)
			if entry.SpecialThanks {
				fmt.Printf("   - Special Thanks: @%v\n", entry.User.Login)
			}
		}
		fmt.Println()
	}
}
