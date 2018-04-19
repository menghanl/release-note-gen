package ghclient

import (
	"net/http"

	"github.com/google/go-github/github"
)

// Client is a github client used to get info from github.
type Client struct {
	owner string
	repo  string

	c *github.Client
}

// GetMergedPRsForMilestone returns a list of github issues that are merged PRs for this milestone.
func GetMergedPRsForMilestone(tc *http.Client, owner, repo, milestone string) []*github.Issue {
	c := &Client{
		owner: owner,
		repo:  repo,
		c:     github.NewClient(tc),
	}
	return c.getMergedPRsForMilestone(milestone)
}
