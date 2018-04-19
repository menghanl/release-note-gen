package notes

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/go-github/github"
)

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

func (c *client) getMergedPRs(issues []*github.Issue) (prs []*github.Issue) {
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
		prs = append(prs, ii)
	}
	return
}

func (c *client) getMergedPRsForMilestone(milestoneTitle string) (prs []*github.Issue) {
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
