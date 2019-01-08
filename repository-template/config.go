package repositoryTemplate

import (
	"context"
	"github.com/google/go-github/v19/github"
	"golang.org/x/oauth2"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

type Config struct {
	GitHubToken string
}

type Client struct {
	GitHubClient      *github.Client
	GitHubGitAuth     transport.AuthMethod
	CommitAuthorEmail string
	CommitAuthorName  string
	CommitMessage     string
}

func (c *Config) NewClient() *Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: c.GitHubToken},
	)
	oauth2Client := oauth2.NewClient(context.Background(), ts)

	return &Client{
		GitHubClient: github.NewClient(oauth2Client),
		GitHubGitAuth: &http.BasicAuth{
			Username: "user", // This can be anything except an empty string
			Password: c.GitHubToken,
		},
	}
}
