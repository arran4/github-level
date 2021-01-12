package main

import (
	"context"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	github_level "github.com/arran4/github-level"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()

	stats := github_level.GetStats(ctx)

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{
			AccessToken: os.Getenv("GITHUB_TOKEN"),
			TokenType:   "bearer",
		},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	githubUser := os.Getenv("GITHUB_REPOSITORY_OWNER")

	user, _, err := client.Users.Get(ctx, githubUser)
	if err != nil {
		log.Panicf("Error: %v", err)
	}

	masterReadmeContents, _, _, err := client.Repositories.GetContents(ctx, githubUser, "github-level", "README.md", &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Panicf("Readme get fail: %v", err)
	}
	if masterReadmeContents.Content == nil {
		log.Panicf("Readme was nil: %v", err)
	}
	c := *masterReadmeContents.Content
	switch masterReadmeContents.GetEncoding() {
	case "base64":
		b, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			log.Panicf("Error %v", err)
		}
		c = string(b)
	}
	presha := sha1.Sum([]byte(c))
	c = ReplaceContent(stats, c)
	postsha := sha1.Sum([]byte(c))
	if presha != postsha {
		_, _, err = client.Repositories.CreateFile(ctx, githubUser, "github-level", "README.md", &github.RepositoryContentFileOptions{
			Message:   github.String("Version Update!"),
			Content:   []byte(c),
			SHA:       masterReadmeContents.SHA,
			Branch:    github.String("main"),
			Committer: &github.CommitAuthor{Name: github.String("Automated " + github_level.PS(user.Name)), Email: user.Email},
		})
		if err != nil {
			log.Panicf("Error creating/updating readme: %v", err)
		}
	} else {
		log.Printf("Presha %v matches post sha %v skipping", presha, postsha)
	}
	if stats.SelfNamedRepo {
		log.Printf("Running in self made profile.")
		RunInSelfNamedRepo(ctx, client, stats, user, githubUser)
	}

}

func RunInSelfNamedRepo(ctx context.Context, client *github.Client, stats *github_level.Stats, user *github.User, githubUser string) {
	userGht := os.Getenv("USER_GITHUB_TOKEN")
	if len(userGht) > 0 {
		log.Printf("Found provided user github token using")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{
				AccessToken: userGht,
				TokenType:   "bearer",
			},
		)
		tc := oauth2.NewClient(ctx, ts)
		client = github.NewClient(tc)
	}

	masterReadmeContents, _, _, err := client.Repositories.GetContents(ctx, githubUser, githubUser, "README.md", &github.RepositoryContentGetOptions{})
	if err != nil {
		log.Panicf("Readme user get fail: %v", err)
	}
	if masterReadmeContents.Content == nil {
		log.Panicf("Readme user was nil: %v", err)
	}
	c := *masterReadmeContents.Content
	switch masterReadmeContents.GetEncoding() {
	case "base64":
		b, err := base64.StdEncoding.DecodeString(c)
		if err != nil {
			log.Panicf("Error user %v", err)
		}
		c = string(b)
	}
	presha := sha1.Sum([]byte(c))
	c = ReplaceContent(stats, c)
	postsha := sha1.Sum([]byte(c))
	c = ReplaceContent(stats, c)
	branch := fmt.Sprintf("githublevel-%s", time.Now().Format("D20060102T1504"))
	if presha != postsha {
		_, _, err = client.Repositories.CreateFile(ctx, githubUser, githubUser, "README.md", &github.RepositoryContentFileOptions{
			Message:   github.String("Version Update!"),
			Content:   []byte(c),
			SHA:       masterReadmeContents.SHA,
			Branch:    github.String(branch),
			Committer: &github.CommitAuthor{Name: github.String("Automated " + github_level.PS(user.Name)), Email: user.Email},
		})
		if err != nil {
			log.Panicf("Error user creating/updating readme: %v", err)
		}
	} else {
		log.Printf("Presha %v matches post sha %v skipping", presha, postsha)
	}

}

func ReplaceContent(stats *github_level.Stats, c string) string {
	shieldLines := make([]string, 0, 1)
	for _, l := range []github_level.Level{
		stats.V1(),
	} {
		shieldLines = append(shieldLines, l.Shield())
	}

	mrc := regexp.MustCompile("\r\n|\r|\n").Split(c, -1)
	nrc := make([]string, 0, len(mrc)+2)
	for _, line := range mrc {
		if strings.Contains(line, "id=\"githubLevelId\"") {
			nrc = append(nrc, shieldLines...)
			shieldLines = make([]string, 0, 0)
		} else {
			nrc = append(nrc, line)
		}
	}
	nrc = append(nrc, shieldLines...)
	return strings.Join(nrc, "\n")
}
