package main

import (
	"fmt"
	"os"
)

type config struct {
	hetznerToken string

	githubToken      string
	githubRepository string
	githubWebhookKey string

	sizeAMD64 string
	sizeARM64 string

	port string
}

const envPrefix = "NUNC_"

func newConfig() (config, error) {
	hetznerToken := os.Getenv(envPrefix + "HETZNER_TOKEN")
	if hetznerToken == "" {
		return config{}, fmt.Errorf("env NUNC_HETZNER_TOKEN is required")
	}

	githubToken := os.Getenv(envPrefix + "GITHUB_TOKEN")
	if githubToken == "" {
		return config{}, fmt.Errorf("env NUNC_GITHUB_TOKEN is required")
	}

	githubRepository := os.Getenv(envPrefix + "GITHUB_REPOSITORY")
	if githubRepository == "" {
		return config{}, fmt.Errorf("env NUNC_GITHUB_REPOSITORY is required")
	}

	githubWebhookKey := os.Getenv(envPrefix + "GITHUB_WEBHOOK_KEY")
	if githubWebhookKey == "" {
		return config{}, fmt.Errorf("env NUNC_GITHUB_WEBHOOK_KEY is required")
	}

	sizeAMD64 := os.Getenv(envPrefix + "INSTANCE_X86")
	if sizeAMD64 == "" {
		return config{}, fmt.Errorf("env NUNC_INSTANCE_X86 is required")
	}

	sizeARM64 := os.Getenv(envPrefix + "INSTANCE_ARM64")
	if sizeARM64 == "" {
		return config{}, fmt.Errorf("env NUNC_INSTANCE_ARM64 is required")
	}

	port := os.Getenv(envPrefix + "HTTP_PORT")
	if port == "" {
		port = "8000"
	}

	return config{
		hetznerToken: hetznerToken,

		githubToken:      githubToken,
		githubRepository: githubRepository,
		githubWebhookKey: githubWebhookKey,

		sizeAMD64: sizeAMD64,
		sizeARM64: sizeARM64,

		port: port,
	}, nil
}
