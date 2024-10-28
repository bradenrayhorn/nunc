package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
)

type githubClient struct {
	client *github.Client
	owner  string
	repo   string
}

func newGithubClient(token string, repository string) (*githubClient, error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("malformed repository path: %s", repository)
	}

	client := github.NewClient(nil).WithAuthToken(token)
	return &githubClient{
		client: client,
		owner:  parts[0],
		repo:   parts[1],
	}, nil
}

func (g *githubClient) getRunners() ([]*github.Runner, error) {
	runners, _, err := g.client.Actions.ListRunners(context.Background(), g.owner, g.repo, &github.ListRunnersOptions{})
	if err != nil {
		return nil, err
	}

	return runners.Runners, nil
}

func (g *githubClient) deleteRunner(id int64) error {
	_, err := g.client.Actions.RemoveRunner(context.Background(), g.owner, g.repo, id)

	return err
}

func (g *githubClient) runnerToken() (string, error) {
	token, _, err := g.client.Actions.CreateRegistrationToken(context.Background(), g.owner, g.repo)

	if err != nil {
		return "", err
	}

	return token.GetToken(), nil
}
