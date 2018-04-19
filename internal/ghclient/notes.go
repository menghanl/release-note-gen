// Package ghclient does something.
package ghclient

import (
	"net/http"

	"github.com/google/go-github/github"
)

// GetMergedPRsForMilestone returns a list of github issues that are merged PRs for this milestone.
func GetMergedPRsForMilestone(tc *http.Client, owner, repo, milestone string) []*github.Issue {
	c := &client{
		owner: owner,
		repo:  repo,
		c:     github.NewClient(tc),
	}
	return c.getMergedPRsForMilestone(milestone)
}
