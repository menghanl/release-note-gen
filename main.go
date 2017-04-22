// release-note-gen generates release notes for a release from closed github PRs.
// It uses all closed PRs labelled as part of the release to generate the note.
//
// For each closed PR, it generates one line release
// note in for form of:
//  description (#<pr>)
// For example:
//  Add grpc.Version string and use it in the UA (#1144)
//
// It uses the PR labels other than release number as the category of the change.
//
//  If a PR has label ["1.3", "API change"], it will be classified as "API change".
//
//  If a PR has more than one labels, the labels will be sorted in the order of
//  "API change", "New Feature", "Behavior change", "Bug fix", "Performance", "Documentation"
//
//  If a PR has only the release number label, it will be classified as "Bug fix".
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

const (
	owner = "grpc"
	repo  = "grpc-go"
)

var (
	noteRegexp = regexp.MustCompile(`^.*\(#[0-9]{4}\)$`)
	token      = flag.String("token", "", "github token")
	release    = flag.String("release", "1.3", "release number")
)

///////////////////// string utils ////////////////////////

func issueToString(ii *github.Issue) string {
	return fmt.Sprintf("%v [%v] - %v\n%v", ii.GetNumber(), ii.GetState(), ii.GetTitle(), ii.GetHTMLURL())
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

func (c *client) getClosedIssuesWithLabel(ctx context.Context, label string) ([]*github.Issue, error) {
	issues, _, err := c.c.Issues.ListByRepo(ctx, owner, repo,
		&github.IssueListByRepoOptions{
			State:  "closed",
			Labels: []string{label},
		},
	)
	if err != nil {
		return nil, err
	}
	return issues, nil
}

func (c *client) getMergeEventForPR(ctx context.Context, issue *github.Issue) (*github.IssueEvent, error) {
	events, _, err := c.c.Issues.ListIssueEvents(ctx, owner, repo, issue.GetNumber(), nil)
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
	cmt, _, err := c.c.Repositories.GetCommit(ctx, owner, repo, ie.GetCommitID())
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
		fmt.Println(ii.Labels)
		fmt.Println(issueToString(ii))
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
		fmt.Println(issueEventToString(ie))
		c, err := c.getCommitFromMerge(ctx, ie)
		if err != nil {
			fmt.Println("failed to get commit message: ", err)
			continue
		}
		prs = append(prs, &mergedPR{issue: ii, commit: c})
	}
	return
}

func getMergedPRsForLabel(httpClient *http.Client, label string) (prs []*mergedPR) {
	c := &client{c: github.NewClient(httpClient)}
	issues, err := c.getClosedIssuesWithLabel(context.Background(), label)
	if err != nil {
		fmt.Println("failed to get issues: ", err)
		return
	}
	prs = c.getMergedPRs(issues)
	return
}

////////////////////////////////////////////////////

///////////////////// pick most weighted label ////////////////////////

var sortWeight = map[string]int{
	"API change":      60,
	"New Feature":     50,
	"Behavior change": 40,
	"Bug fix":         30,
	"Performance":     20,
	"Documentation":   10,
}

func sortLabelName(labels []string) []string {
	sort.Slice(labels, func(i, j int) bool {
		return sortWeight[labels[i]] >= sortWeight[labels[j]]
	})
	return labels
}

func pickMostWeightedLabel(labels []github.Label) string {
	var names []string
	for _, l := range labels {
		names = append(names, l.GetName())
	}
	sortLabelName(names)
	if sortWeight[names[0]] == 0 {
		return "Bug fix" // Default to bug fix
	}
	return names[0]
}

////////////////////////////////////////////////////

///////////////////// generate notes ////////////////////////

func generateNotes(prs []*mergedPR) (notes map[string][]string) {
	fmt.Print("\n================ generating notes ================\n\n")
	notes = make(map[string][]string)
	for _, pr := range prs {
		label := pickMostWeightedLabel(pr.issue.Labels)
		n := getFirstLine(pr.commit.GetMessage())
		if ok := noteRegexp.MatchString(n); !ok {
			fmt.Println("   ++++ doesn't match noteRegexp, ", n)
			n = fmt.Sprintf("%v (#%d)", pr.issue.GetTitle(), pr.issue.GetNumber())
		}
		fmt.Println(n)
		notes[label] = append(notes[label], n)
	}
	return
}

////////////////////////////////////////////////////

func main() {
	flag.Parse()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)

	prs := getMergedPRsForLabel(tc, *release)
	notes := generateNotes(prs)
	fmt.Printf("\n================ generated notes for release %v ================\n\n", *release)

	var keys []string
	for k := range notes {
		keys = append(keys, k)
	}
	sortLabelName(keys)
	for _, k := range keys {
		fmt.Println("#", k)
		for _, n := range notes[k] {
			fmt.Println(" *", n)
		}
	}
}
