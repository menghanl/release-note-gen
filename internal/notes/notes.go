// Package notes pulls PR from github and generate release notes.
package notes

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-github/github"
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

// Note is one release note element.
type Note struct {
	Title string
	Sub   []string
}

func (n *Note) toMarkdown(includeSub bool) string {
	ret := " * " + n.Title
	if includeSub {
		for _, s := range n.Sub {
			ret += "\n   - " + s
		}
	}
	return ret
}

func generateNotes(prs []*github.Issue, grpcMembers, urwelcomeMap, verymuchMap map[string]struct{}) (notes map[string][]*Note) {
	fmt.Print("\n================ generating notes ================\n\n")
	notes = make(map[string][]*Note)
	for _, pr := range prs {
		label := pickMostWeightedLabel(pr.Labels)
		_, ok := labelToSectionName[label]
		if !ok {
			continue // If ok==false, ignore this PR in the release note.
		}
		fmt.Printf(" [%v] - ", color.BlueString("%v", pr.GetNumber()))
		fmt.Print(color.GreenString("%-18q", label))
		fmt.Printf(" from: %v\n", labelsToString(pr.Labels))

		noteLine := &Note{
			Title: fmt.Sprintf("%v (#%d)", pr.GetTitle(), pr.GetNumber()),
		}

		user := pr.GetUser().GetLogin()

		_, isGRPCMember := grpcMembers[user]
		_, isWelcome := urwelcomeMap[user]
		_, isVerymuch := verymuchMap[user]

		if isVerymuch || (!isGRPCMember && !isWelcome) {
			noteLine.Sub = append(noteLine.Sub, "Special thanks: "+"@"+user)
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
	c := &client{c: github.NewClient(tc)}

	prs := c.getMergedPRsForMilestone(*release + milestoneTitleSurfix)

	var (
		urwelcomeMap map[string]struct{}
		verymuchMap  map[string]struct{}
		grpcMembers  map[string]struct{}
	)
	if *thanks {
		urwelcomeMap = commaStringToSet(*urwelcome)
		verymuchMap = commaStringToSet(*verymuch)
		grpcMembers = c.getOrgMembers("grpc")
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
