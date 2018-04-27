package main

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/menghanl/release-note-gen/ghclient"
	"golang.org/x/oauth2"
)

func main() {
	var tc *http.Client
	token := "dcfe7dcf8afea7d469450bd4ad2152d32a3128c1"
	if token != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(ctx, ts)
	}

	c := ghclient.New(tc, "grpc", "grpc-go")
	prs := c.GetMergedPRsForMilestone("1.12 Release")

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/release", func(c *gin.Context) {
		c.JSON(200, prs)
	})
	r.Run() // listen and serve on 0.0.0.0:8080
}
