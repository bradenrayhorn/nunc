package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"
)

func main() {
	slog.Info("starting")

	config, err := newConfig()
	if err != nil {
		panic(err)
	}

	// start webhook listener

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	hetzner, err := newHetzner(config.hetznerToken)
	if err != nil {
		panic(err)
	}

	githubClient, err := newGithubClient(config.githubToken, config.githubRepository)
	if err != nil {
		panic(err)
	}

	state := &serverState{
		runners: make(map[string]runner, 0),
	}

	startCleanupJob(state, hetzner, githubClient)

	handler := newHttpHandler(config, state, hetzner, githubClient)
	go http.ListenAndServe(":"+config.port, handler)

	<-c
	slog.Info("shutting down")
}

func startCleanupJob(state *serverState, hetzner *hetzner, githubClient *githubClient) {

	go func() {
		for {
			err := doCleanup(state, hetzner, githubClient)
			if err != nil {
				slog.Error("clean job failure", "error", err)
			}

			time.Sleep(time.Minute * 1)
		}
	}()
}

func doCleanup(state *serverState, hetzner *hetzner, githubClient *githubClient) error {
	servers, err := hetzner.listServers()
	if err != nil {
		return fmt.Errorf("hetzner servers: %w", err)
	}

	runners, err := githubClient.getRunners()
	if err != nil {
		return fmt.Errorf("github runners: %w", err)
	}

	state.mutex.Lock()
	defer state.mutex.Unlock()

	for id, runner := range state.runners {
		fmt.Printf("checking active runner %s, status = %s\n", runner.name, runner.status)
		// if server is dead, remove it
		if runner.status == serverStatusDead {
			slog.Info("removing dead runner", "id", id)
			delete(state.runners, id)
		}

		// delete working server is the job began more than 55 minutes ago - it must be stuck
		if runner.status == serverStatusWorking && runner.lastJobBeganAt.Sub(time.Now()).Abs() > (time.Minute*55) {
			slog.Info("removing runner working for more than 55 minutes", "id", id)
			delete(state.runners, id)
		}

		// delete idle server if it is approaching any intervals of one hour after creation
		//   servers bill a minimum of one hour, so this maximizes usage for cost
		if runner.status == serverStatusIdle {
			limit := runner.createdAt.Add(60 * time.Minute)

			for {
				fmt.Printf("limit = %v\n", limit)
				if limit.Compare(time.Now()) < 0 {
					limit = limit.Add(60 * time.Minute)
				} else {
					break
				}
			}

			fmt.Printf("idle diff = %v \n", limit.Sub(time.Now()))
			if limit.Sub(time.Now()) < time.Minute*5 {
				slog.Info("removing idle runner, approaching limit",
					"id", id,
					"limit", limit,
					"createdAt", runner.createdAt,
				)
				delete(state.runners, id)
			}
		}
	}

	// delete unknown servers
	for _, sv := range servers {
		if _, ok := state.runners[sv.Name]; !ok {
			slog.Info("deleting server",
				"name", sv.Name,
			)

			if err := hetzner.deleteServer(sv); err != nil {
				return fmt.Errorf("hetzner delete server: %w", err)
			}
		}
	}

	// delete unknown github runners
	for _, runner := range runners {
		if _, ok := state.runners[runner.GetName()]; !ok {
			slog.Info("deleting github runner",
				"name", runner.GetName(),
			)

			if err := githubClient.deleteRunner(runner.GetID()); err != nil {
				return fmt.Errorf("github delete runner: %w", err)
			}
		}
	}

	return nil
}
