package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/google/go-github/v66/github"
)

func newHttpHandler(config config, state *serverState, hetzner *hetzner, githubClient *githubClient) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	mux.HandleFunc("/github-webhook", func(w http.ResponseWriter, r *http.Request) {
		err := handleCallback(config, state, hetzner, githubClient, r)
		if err != nil {
			slog.Error("github webhook error", "error", err)

			w.WriteHeader(500)
		}
	})

	return mux
}

func handleCallback(
	config config,
	state *serverState,
	hetzner *hetzner,
	githubClient *githubClient,
	r *http.Request,
) error {
	payload, err := github.ValidatePayload(r, []byte(config.githubWebhookKey))
	if err != nil {
		return err
	}

	anyEvent, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		return err
	}

	if event, ok := anyEvent.(*github.WorkflowJobEvent); ok {
		labels := event.GetWorkflowJob().Labels
		slog.Info("received webhook", "action", event.GetAction(), "labels", labels)

		if event.GetAction() != "queued" {
			return nil
		}

		if !slices.Contains(labels, "self-hosted") {
			return nil
		}

		var arch string
		if slices.Contains(labels, archARM64) {
			arch = archARM64
		} else if slices.Contains(labels, archAMD64) {
			arch = archAMD64
		} else {
			slog.Error("got webhook with unknown arch", "labels", labels)
			return nil
		}

		state.mutex.Lock()
		defer state.mutex.Unlock()

		// check if a free runner already exists
		var existingRunner *runner
		for _, runner := range state.runners {
			if runner.arch == arch && runner.status == serverStatusIdle {
				existingRunner = &runner
			}
		}

		if existingRunner != nil {
			slog.Info("reusing existing runner", "id", existingRunner.name)
			runner := *existingRunner
			runner.status = serverStatusWorking
			runner.lastJobBeganAt = time.Now()
			state.runners[runner.name] = runner

			connectAndRunWorker(runner, state, hetzner, githubClient)
			return nil
		}

		// need to make a new runner
		runnersWithArch := 0
		for _, runner := range state.runners {
			if runner.arch == arch {
				runnersWithArch += 1
			}
		}

		// find the id and size
		id := fmt.Sprintf("%s-%d", arch, runnersWithArch)
		slog.Info("creating new runner", "id", id)

		size := "unknown"
		if arch == archARM64 {
			size = config.sizeARM64
		} else if arch == archAMD64 {
			size = config.sizeAMD64
		}

		// create server
		server, err := hetzner.newServer(context.Background(), size, id)
		if err != nil {
			return fmt.Errorf("create hetzner server: %w", err)
		}

		// persist and start worker
		state.runners[id] = runner{
			name: id,
			arch: arch,

			server: server,
			status: serverStatusWorking,

			lastJobBeganAt: time.Now(),
			createdAt:      time.Now(),
		}

		connectAndRunWorker(state.runners[id], state, hetzner, githubClient)

		return nil
	} else {
		return fmt.Errorf("invalid workflow event: %T", event)
	}
}
