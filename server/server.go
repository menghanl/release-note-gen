package main

import (
	"context"
	"flag"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/go-github/github"
	"github.com/menghanl/release-note-gen/ghclient"
	"github.com/menghanl/release-note-gen/notes"
	"golang.org/x/oauth2"
)

var token = flag.String("token", "", "github token")

func main() {
	var tc *http.Client
	if *token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	c := ghclient.New(tc, "grpc", "grpc-go")
	prs := c.GetMergedPRsForMilestone("1.12 Release")

	ns := notes.GenerateNotes(prs, notes.Filters{
		SpecialThanks: func(pr *github.Issue) bool {
			return false
		},
	})

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/release", func(c *gin.Context) {
		c.JSON(200, ns)
	})
	r.Run() // listen and serve on 0.0.0.0:8080
}
