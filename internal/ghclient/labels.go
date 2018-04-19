package ghclient

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/go-github/github"
)

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