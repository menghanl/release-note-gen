// Package notes defines the structs for release note and functions to generate
// notes.
package notes

// Notes contains all the note entries for a given release.
type Notes struct {
	Org     string
	Repo    string
	Version string
	Entries map[string][]Entry
}

// Entry contains the info for one entry in the release notes.
type Entry struct {
	ID      int64
	Number  int
	Title   string
	HTMLURL string

	User      *User
	MileStone *MileStone

	SpecialThanks bool
}

// User represents a github user.
type User struct {
	AvatarURL string
	HTMLURL   string
	Login     string
}

// MileStone represents a github milestone.
type MileStone struct {
	ID    int64
	Title string
}

// Label represents a github label.
type Label struct {
	Name string
}
