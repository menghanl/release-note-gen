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
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	noteRegexp = regexp.MustCompile(`^.*\(#[0-9]+\)$`)
	token      = flag.String("token", "", "github token")
	release    = flag.String("release", "", "release number")
	owner      = flag.String("owner", "grpc", "github repo owner")
	repo       = flag.String("repo", "grpc-go", "github repo")
	thanks     = flag.Bool("thanks", false, "whether to include thank you note. grpc organization members are excluded")
	urwelcome  = flag.String("urwelcome", "", "list of users to exclude from thank you note, format: user1,user2")
)

///////////////////// string utils ////////////////////////

func issueToString(ii *github.Issue) string {
	var ret string
	ret += color.CyanString("%v [%v] - %v", ii.GetNumber(), ii.GetState(), ii.GetTitle())
	ret += "\n - "
	ret += color.BlueString("%v", ii.GetHTMLURL())
	ret += "\n - "
	ret += color.BlueString("author: %v", *ii.GetUser().Login)
	return ret
}

func labelsToString(ls []github.Label) string {
	var names []string
	for _, l := range ls {
		names = append(names, l.GetName())
	}
	return fmt.Sprintf("%v", names)
}

func issueEventToString(ie *github.IssueEvent) string {
	return fmt.Sprintf("[%v]", ie.GetEvent())
}

func getFirstLine(s string) string {
	ss := strings.Split(s, "\n")
	return ss[0]
}

////////////////////////////////////////////////////

///////////////////// get PR for label ////////////////////////

type client struct {
	c *github.Client
}

func (c *client) getMilestoneNumberForTitle(ctx context.Context, milestoneTitle string) (int, error) {
	fmt.Println("milestone title: ", milestoneTitle)
	milestones, _, err := c.c.Issues.ListMilestones(context.Background(), *owner, *repo,
		&github.MilestoneListOptions{
			State:       "all",
			ListOptions: github.ListOptions{PerPage: 100},
		},
	)
	if err != nil {
		return 0, err
	}
	fmt.Println("count milestones", len(milestones))
	for _, m := range milestones {
		if m.GetTitle() == milestoneTitle {
			return m.GetNumber(), nil
		}
	}
	return 0, fmt.Errorf("no milestone with title %q was found", milestoneTitle)
}

func (c *client) getClosedIssuesWithMilestoneNumber(ctx context.Context, milestoneNumber string) ([]*github.Issue, error) {
	fmt.Println("milestone number: ", milestoneNumber)
	issues, _, err := c.c.Issues.ListByRepo(ctx, *owner, *repo,
		&github.IssueListByRepoOptions{
			State:       "closed",
			Milestone:   milestoneNumber,
			ListOptions: github.ListOptions{PerPage: 1000},
		},
	)
	if err != nil {
		return nil, err
	}
	fmt.Println("count issues", len(issues))
	return issues, nil
}

func (c *client) getMergeEventForPR(ctx context.Context, issue *github.Issue) (*github.IssueEvent, error) {
	events, _, err := c.c.Issues.ListIssueEvents(ctx, *owner, *repo, issue.GetNumber(), nil)
	if err != nil {
		return nil, err
	}
	for _, e := range events {
		if e.GetEvent() == "merged" {
			return e, nil
		}
	}
	return nil, fmt.Errorf("merge event not found")
}

func (c *client) getCommitFromMerge(ctx context.Context, ie *github.IssueEvent) (*github.Commit, error) {
	if ie.GetEvent() != "merged" {
		return nil, fmt.Errorf("not merge issue event")
	}
	cmt, _, err := c.c.Repositories.GetCommit(ctx, *owner, *repo, ie.GetCommitID())
	if err != nil {
		return nil, err
	}
	return cmt.Commit, err
}

type mergedPR struct {
	issue  *github.Issue
	commit *github.Commit
}

func (c *client) getMergedPRs(issues []*github.Issue) (prs []*mergedPR) {
	ctx := context.Background()
	for _, ii := range issues {
		fmt.Println(issueToString(ii))
		fmt.Println(" -", labelsToString(ii.Labels))
		if ii.PullRequestLinks == nil {
			fmt.Println("not a pull request")
			continue
		}
		// ii is a PR.
		ie, err := c.getMergeEventForPR(ctx, ii)
		if err != nil {
			fmt.Println("failed to get merge event: ", err)
			continue
		}
		fmt.Println(" -", issueEventToString(ie))
		c, err := c.getCommitFromMerge(ctx, ie)
		if err != nil {
			fmt.Println("failed to get commit message: ", err)
			continue
		}
		prs = append(prs, &mergedPR{issue: ii, commit: c})
	}
	return
}

func (c *client) getMergedPRsForMilestone(milestoneTitle string) (prs []*mergedPR) {
	num, err := c.getMilestoneNumberForTitle(context.Background(), milestoneTitle)
	if err != nil {
		fmt.Println("failed to get milestone number: ", err)
	}
	issues, err := c.getClosedIssuesWithMilestoneNumber(context.Background(), strconv.Itoa(num))
	if err != nil {
		fmt.Println("failed to get issues: ", err)
		return
	}
	return c.getMergedPRs(issues)
}

func (c *client) getOrgMembers(org string) map[string]struct{} {
	opt := &github.ListMembersOptions{}
	var count int
	ret := make(map[string]struct{})
	for {
		members, resp, err := c.c.Organizations.ListMembers(context.Background(), org, opt)
		if err != nil {
			fmt.Println("failed to get org members: ", err)
			return nil
		}
		for _, m := range members {
			ret[m.GetLogin()] = struct{}{}
		}
		count += len(members)
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}
	fmt.Printf("%v members in org %v\n", count, org)
	return ret
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

func generateNotes(prs []*mergedPR, grpcMembers, urwelcomeMap map[string]struct{}) (notes map[string][]*note) {
	fmt.Print("\n================ generating notes ================\n\n")
	notes = make(map[string][]*note)
	for _, pr := range prs {
		label := pickMostWeightedLabel(pr.issue.Labels)
		_, ok := labelToSectionName[label]
		if !ok {
			continue // If ok==false, ignore this PR in the release note.
		}
		fmt.Printf(" [%v] - ", color.BlueString("%v", pr.issue.GetNumber()))
		fmt.Print(color.GreenString("%-18q", label))
		fmt.Printf(" from: %v\n", labelsToString(pr.issue.Labels))

		n := getFirstLine(pr.commit.GetMessage())
		if ok := noteRegexp.MatchString(n); !ok {
			color.Red("   ++++ doesn't match noteRegexp, ", n)
			n = fmt.Sprintf("%v (#%d)", pr.issue.GetTitle(), pr.issue.GetNumber())
		}
		noteLine := &note{
			head: n,
		}

		user := pr.issue.GetUser().GetLogin()
		if _, ok := grpcMembers[user]; !ok {
			if _, ok := urwelcomeMap[user]; !ok {
				noteLine.sub = append(noteLine.sub, "Special thanks: "+"@"+user)
			}
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
	c := &client{c: github.NewClient(tc)}

	prs := c.getMergedPRsForMilestone(*release + milestoneTitleSurfix)

	var (
		urwelcomeMap map[string]struct{}
		grpcMembers  map[string]struct{}
	)
	if *thanks {
		urwelcomeMap = make(map[string]struct{})
		tmp := strings.Split(*urwelcome, ",")
		for _, t := range tmp {
			urwelcomeMap[t] = struct{}{}
		}
		grpcMembers = c.getOrgMembers("grpc")
	}
	notes := generateNotes(prs, grpcMembers, urwelcomeMap)

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
