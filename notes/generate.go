package notes

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/google/go-github/github"
)

// Filters contains filters to be applied on input PRs.
type Filters struct {
	// If Ignore returns true, the pr will be excluded from the notes.
	Ignore func(pr *github.Issue) bool
	// if SpecialThanks returns true, a special thanks note will be included for
	// the author.
	SpecialThanks func(pr *github.Issue) bool
}

// GenerateNotes generate the release notes from the given prs and maps.
func GenerateNotes(prs []*github.Issue, filters Filters) *Notes {
	notes := Notes{
		Org:     "todo",
		Repo:    "todo",
		Version: "todo",
	}

	sections := make(map[string]*Section)

	for _, pr := range prs {
		label := pickMostWeightedLabel(pr.Labels)
		_, ok := labelToSectionName[label]
		if !ok {
			continue // If ok==false, ignore this PR in the release note.
		}
		fmt.Printf(" [%v] - ", color.BlueString("%v", pr.GetNumber()))
		fmt.Print(color.GreenString("%-18q", label))
		fmt.Printf(" from: %v\n", labelsToString(pr.Labels))

		section, ok := sections[label]
		if !ok {
			section = &Section{Name: label}
			sections[label] = section
		}

		user := pr.GetUser()

		entry := &Entry{
			// head: fmt.Sprintf("%v (#%d)", pr.GetTitle(), pr.GetNumber()),
			IssueNumber: pr.GetNumber(),
			Title:       pr.GetTitle(),
			HTMLURL:     pr.GetHTMLURL(),

			User: &User{
				AvatarURL: user.GetAvatarURL(),
				HTMLURL:   user.GetHTMLURL(),
				Login:     user.GetLogin(),
			},

			// MileStone: &MileStone{
			// }

			SpecialThanks: false,
		}
		section.Entries = append(section.Entries, entry)
	}

	return &notes
}
