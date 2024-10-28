# nunc

Small service to automatically deploy self-hosted GitHub runners on-demand via Hetzner server hosting.

## Use case

As of creation, GitHub does not provide arm64 based runners individuals using GitHub actions.

Using GitHub-hosted runners to build an arm64 variant of a Docker image requires emulation on the
x86 GitHub-hosted runners which can be slow.

A self-hosted arm64 worker can be connected to a repository, but paying to keep the server running
24/7 can be expensive for non-frequent use.

nunc will listen to GitHub workflow events and create a new Hetzner server on-demand when a workflow
job requests one. The server will be destroyed after the job completes. Depending on the server instance
required, this could cost as little as 0.01 Euros per job run.

