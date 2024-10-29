package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"log/slog"
	"math"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshKey struct {
	pubKeyString string
	key          *rsa.PrivateKey
}

func newSSHKey() (sshKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return sshKey{}, fmt.Errorf("generate key: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(&key.PublicKey)
	if err != nil {
		return sshKey{}, fmt.Errorf("ssh key: %w", err)
	}

	pubKeyString := fmt.Sprintf("ssh-rsa %s", base64.StdEncoding.EncodeToString(sshPubKey.Marshal()))

	return sshKey{
		pubKeyString: pubKeyString,
		key:          key,
	}, nil
}

func connectAndRunWorker(runner runner, state *serverState, hetzner *hetzner, githubClient *githubClient) {
	maxAttempts := 10

	go func() {
		// if the server is brand new, wait a little bit for the ssh server to be ready
		if time.Since(runner.createdAt) < 11*time.Second {
			time.Sleep(11 * time.Second)
		}

		attempts := 0

		for {
			// break out after max attempts
			if attempts > maxAttempts {
				slog.Error("could not start up worker, giving up")
				break
			}

			// try to ssh into server
			err := attemptConnectAndRunWorker(runner, hetzner.key, githubClient)
			if err != nil {
				attempts += 1
				slog.Error("failed to start worker", "error", err, "attempts", attempts)

				time.Sleep((175 * time.Millisecond) * time.Duration(math.Pow(2, float64(attempts))))
				continue
			}

			break
		}

		// release runner
		slog.Info("rebuilding server", "id", runner.name)
		err := hetzner.reimageServer(context.Background(), runner.server)
		if err != nil {
			slog.Error("could not rebuild server", "error", err)
			releaseRunner(runner.name, state, serverStatusDead)
		} else {
			slog.Info("server rebuilt", "id", runner.name)
			releaseRunner(runner.name, state, serverStatusIdle)
		}
	}()
}

func releaseRunner(id string, state *serverState, status string) {
	state.mutex.Lock()
	defer state.mutex.Unlock()

	if runner, ok := state.runners[id]; ok {
		runner.status = status
		state.runners[id] = runner
	}

}

const runWorkerBash = `mkdir actions-runner && cd actions-runner && \
curl -o actions-runner-linux-%s-2.320.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.320.0/actions-runner-linux-%s-2.320.0.tar.gz && \
tar xzf ./actions-runner-linux-%s-2.320.0.tar.gz && \
./config.sh --unattended --url https://github.com/%s --token %s --name %s --no-default-labels --labels self-hosted,%s --ephemeral --disableupdate && \
./run.sh; sudo poweroff
`

func attemptConnectAndRunWorker(runner runner, key sshKey, githubClient *githubClient) error {
	ip := runner.server.PublicNet.IPv4.IP
	id := runner.name
	arch := runner.arch
	repository := githubClient.owner + "/" + githubClient.repo

	token, err := githubClient.runnerToken()
	if err != nil {
		return fmt.Errorf("get runner token from github: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(key.key)
	if err != nil {
		return fmt.Errorf("make signer: %w", err)
	}

	config := &ssh.ClientConfig{
		User:            "gh",
		Timeout:         time.Minute * 30,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	client, err := ssh.Dial("tcp", net.JoinHostPort(ip.String(), "22"), config)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open session: %w", err)
	}

	defer func() { _ = session.Close() }()
	var out bytes.Buffer
	var stdErr bytes.Buffer
	session.Stdout = &out
	session.Stderr = &stdErr

	slog.Info("opened session", "id", id)

	githubArch := "unknown-github-arch"
	if arch == archARM64 {
		githubArch = "arm64"
	} else if arch == archAMD64 {
		githubArch = "x86"
	}
	cmd := fmt.Sprintf(runWorkerBash, githubArch, githubArch, githubArch, repository, token, id, arch)
	err = session.Start(cmd)
	if err != nil {
		return fmt.Errorf("run: %w; out = %s, err = %s", err, out.String(), stdErr.String())
	}

	if err := session.Wait(); err != nil {
		return fmt.Errorf("waiting: %w; out = %s, err = %s", err, out.String(), stdErr.String())
	}

	slog.Info("session complete", "out", out.String(), "err", stdErr.String())

	return nil
}
