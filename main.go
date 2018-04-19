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
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-github/github"
	"github.com/menghanl/release-note-gen/internal/notes"
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

func labelsToString(ls []github.Label) string {
	var names []string
	for _, l := range ls {
		names = append(names, l.GetName())
	}
	return fmt.Sprintf("%v", names)
}

////////////////////////////////////////////////////

///////////////////// pick most weighted label ////////////////////////

const (
	labelPrefix  = "Type: "
	defaultLabel = "Bug"
)

var sortWeight = map[string]int{
	"Dependencies":     70,
	"API Change":       60,
	"Behavior Change":  50,
	"Feature":          40,
	"Performance":      30,
	"Bug":              20,
	"Documentation":    10,
	"Testing":          0,
	"Internal Cleanup": 0,
}

func sortLabelName(labels []string) []string {
	sort.Slice(labels, func(i, j int) bool {
		return sortWeight[labels[i]] >= sortWeight[labels[j]]
	})
	return labels
}

func pickMostWeightedLabel(labels []github.Label) string {
	if len(labels) <= 0 {
		fmt.Println("0 lable was assigned to issue")
		return defaultLabel
	}
	var names []string
	for _, l := range labels {
		names = append(names, strings.TrimPrefix(l.GetName(), labelPrefix))
	}
	sortLabelName(names)
	if _, ok := sortWeight[names[0]]; !ok {
		return defaultLabel
	}
	return names[0]
}

////////////////////////////////////////////////////

///////////////////// generate notes ////////////////////////

type note struct {
	head string
	sub  []string
}

func (n *note) toMarkdown(includeSub bool) string {
	ret := " * " + n.head
	if includeSub {
		for _, s := range n.sub {
			ret += "\n   - " + s
		}
	}
	return ret
}

func generateNotes(prs []*github.Issue, grpcMembers, urwelcomeMap, verymuchMap map[string]struct{}) (notes map[string][]*note) {
	fmt.Print("\n================ generating notes ================\n\n")
	notes = make(map[string][]*note)
	for _, pr := range prs {
		label := pickMostWeightedLabel(pr.Labels)
		_, ok := labelToSectionName[label]
		if !ok {
			continue // If ok==false, ignore this PR in the release note.
		}
		fmt.Printf(" [%v] - ", color.BlueString("%v", pr.GetNumber()))
		fmt.Print(color.GreenString("%-18q", label))
		fmt.Printf(" from: %v\n", labelsToString(pr.Labels))

		noteLine := &note{
			head: fmt.Sprintf("%v (#%d)", pr.GetTitle(), pr.GetNumber()),
		}

		user := pr.GetUser().GetLogin()

		_, isGRPCMember := grpcMembers[user]
		_, isWelcome := urwelcomeMap[user]
		_, isVerymuch := verymuchMap[user]

		if isVerymuch || (!isGRPCMember && !isWelcome) {
			noteLine.sub = append(noteLine.sub, "Special thanks: "+"@"+user)
		}

		notes[label] = append(notes[label], noteLine)
	}
	return
}

////////////////////////////////////////////////////

const milestoneTitleSurfix = " Release" // For example, "1.7 Release".
var labelToSectionName = map[string]string{
	"Dependencies":    "Dependencies",
	"API Change":      "API Changes",
	"Behavior Change": "Behavior Changes",
	"Feature":         "New Features",
	"Performance":     "Performance Improvements",
	"Bug":             "Bug Fixes",
	"Documentation":   "Documentation",
}

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

	var tc *http.Client
	if *token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}
	prs := notes.GetMergedPRsForMilestone(tc, *owner, *repo, *release+milestoneTitleSurfix)

	var (
		urwelcomeMap map[string]struct{}
		verymuchMap  map[string]struct{}
		grpcMembers  map[string]struct{}
	)
	if *thanks {
		urwelcomeMap = commaStringToSet(*urwelcome)
		verymuchMap = commaStringToSet(*verymuch)
		// grpcMembers = c.getOrgMembers("grpc")
	}
	notes := generateNotes(prs, grpcMembers, urwelcomeMap, verymuchMap)

	fmt.Printf("\n================ generated notes for release %v ================\n\n", *release)
	var keys []string
	for k := range notes {
		keys = append(keys, k)
	}
	sortLabelName(keys)
	for _, k := range keys {
		fmt.Println()
		fmt.Println("#", labelToSectionName[k])
		for _, n := range notes[k] {
			fmt.Println(n.toMarkdown(*thanks))
		}
	}
}
