package main

import (
	"context"
	"flag"
	"fmt"
	"regexp"
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
)

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

func (c *client) getCommitMessageFromMerge(ctx context.Context, ie *github.IssueEvent) (string, error) {
	if ie.GetEvent() != "merged" {
		return "", fmt.Errorf("not merge issue event")
	}
	cmt, _, err := c.c.Repositories.GetCommit(ctx, owner, repo, ie.GetCommitID())
	if err != nil {
		return "", err
	}
	return cmt.Commit.GetMessage(), err
}

func (c *client) generateNotes(issues []*github.Issue) (notes []string) {
	ctx := context.Background()
	for _, ii := range issues {
		fmt.Println(issueToString(ii))
		if ii.PullRequestLinks == nil {
			fmt.Println("not a pull request")
			continue
		}
		ie, err := c.getMergeEventForPR(ctx, ii)
		if err != nil {
			fmt.Println("failed to get merge event: ", err)
			continue
		}
		fmt.Println(issueEventToString(ie))
		m, err := c.getCommitMessageFromMerge(ctx, ie)
		if err != nil {
			fmt.Println("failed to get commit message: ", err)
		}
		// fmt.Println(m) // TODO: write a file.
		n := getFirstLine(m)
		if ok := noteRegexp.MatchString(n); !ok {
			fmt.Println("   ++++ doesn't match noteRegexp, ", n)
			n = fmt.Sprintf("%v (#%d)", ii.GetTitle(), ii.GetNumber())
		}
		fmt.Println(n)
		notes = append(notes, n)
		fmt.Println()
	}
	return
}

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

func main() {
	flag.Parse()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(ctx, ts)
	c := &client{c: github.NewClient(tc)}

	ctx = context.Background()
	issues, err := c.getClosedIssuesWithLabel(ctx, "1.3")
	if err != nil {
		fmt.Println("failed to get issues: ", err)
		return
	}
	notes := c.generateNotes(issues)
	fmt.Print(" ================ generated notes ================\n\n")
	for _, n := range notes {
		fmt.Println(n)
	}
}
