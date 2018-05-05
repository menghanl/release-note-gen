package notes

import "github.com/google/go-github/github"

type Filters struct {
	// If Ignore returns true, the pr will be excluded from the notes.
	Ignore func(pr *github.Issue) bool
	// if SpecialThanks returns true, a special thanks note will be included for
	// the author.
	SpecialThanks func(pr *github.Issue) bool
}

// GenerateNotes generate the release notes from the given prs and maps.
func GenerateNotes(prs []*github.Issue, filters Filters) {
}
