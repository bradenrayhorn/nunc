# nunc

Small service to automatically deploy self-hosted GitHub runners on-demand via Hetzner server hosting.

## NOTICE:

This is mostly obsolete now that GitHub is [offering free public](https://github.blog/changelog/2025-01-16-linux-arm64-hosted-runners-now-available-for-free-in-public-repositories-public-preview/) ARM runners.

## Use case

As of creation, GitHub does not provide arm64 based runners for individuals using GitHub actions.

Using GitHub-hosted runners to build an arm64 variant of a Docker image requires emulation on
x86 GitHub-hosted runners which can be slow.

A self-hosted arm64 worker can be connected to a repository, but paying to keep the server running
24/7 can be expensive for infrequent use.

nunc will listen to GitHub workflow events and create a new Hetzner server on-demand when a workflow
job requests one. The server will be destroyed after the job completes. Depending on the server instance
required, this could cost as little as 0.01 Euros per job run.

## Server management

Servers will not be deleted after each job run, rather, they are reformatted so that they can be
re-used for other jobs to minimize costs.

If a server is idle and approaching intervals of 1 hour from when it was created, the server will be
deleted. Hetzner bills in intervals of 1 hour, so this maximizes potential usage for the cost.
For example, if a server is created at 9:13am it may be deleted just before 10:13am, 11:13am, 12:13pm, etc.

## Deployment

A Docker image is provided to easily run nunc. Configuration is done via env variables.

The image is hosted on GHCR: `ghcr.io/bradenrayhorn/nunc`. The `latest` tag or a specific version tag (ex: `0.1.0`) can be used.

### Step by step

1. Create GitHub fine-grained personal access token for the repository
      - Scope the access token to the single repository this instance of nunc will be used for
      - Requires the following repository permissions:
        - Administration: Read and write
        - Metadata: Read-only
      - No user permissions are required
2. Create Hetzner API token for the project that will be used to create servers
      - Requires Read & Write permission.
      - The Hetzner project **MUST** only be used for running a single instance of nunc.
      - _**IMPORTANT WARNING**_: Any other servers found in the project will be _**DELETED!**_
3. Configure and deploy nunc - required env variables are listed in the Configuration section below.
      - The nunc server must be publicly accessibly over HTTP in order to receive callbacks from GitHub
      - nunc provides a `/health` endpoint that can be used as a basic test to check if nunc is running
4. Setup GitHub webook on the repository. This is used to tell nunc when a workflow is running and a new server is needed.
      - Payload URL: `PUBLIC_URL_OF_NUNC/github-webhook`
      - Content type: `application/json`
      - Secret: Must match `NUNC_GITHUB_WEBHOOK_KEY` env variable
      - To select events to subscribe to, choose "Let me select individual events."
        - Only the "Workflow jobs" event is required
5. Setup GitHub workflow.
      - To request a job be run on your Hetzner server, use the `runs-on` syntax on any GitHub workflow job
      - Examples, depending on desired architecture:
        - `runs-on: [self-hosted, arm64]`
        - `runs-on: [self-hosted, amd64]`

### Configuration

| Variable | Description | Required | Default | Values |
| - | - | - | - | - |
| `NUNC_HTTP_PORT` | Port to run on. | No | `8000` | Example: `8080` |
| `NUNC_HETZNER_TOKEN` | Hetzner API token | Yes | | |
| `NUNC_GITHUB_TOKEN` | Github access token | Yes | | |
| `NUNC_GITHUB_REPOSITORY` | GitHub repository to connect to | Yes | | Example: `bradenrayhorn/nunc` |
| `NUNC_GITHUB_WEBHOOK_KEY` | Randomly generated secret key for securing webhooks | Yes | | Example: `ks9M8nVwGeBLACxzr+cSTQ==` |
| `NUNC_INSTANCE_X86` | Hetzner instance to deploy for x86 jobs | Yes | | Example: `cpx31` |
| `NUNC_INSTANCE_ARM64` | Hetzner instance to deploy for arm64 jobs | Yes | | Example: `cax31` |

## Disclaimer

You will be billed by Hetzner for server time used. Please consider how many jobs you will be dispatching
and evaluate the cost before setting up nunc.

Hetzner allows you to set up a usage email notification when the project crosses over a certain Euro
threshold. This is helpful to prevent surprise bills at the end of the month.

## Current limitations

- nunc will lose any state of active servers or job runs upon restarting
  - When starting up, old servers and jobs will be cancelled and deleted
- nunc is not high availability - only one nunc instance can be run per repository at a time
- Jobs can run at most 55 minutes before being terminated - this could be made configurable in the future
- Only x86 and arm64 Linux VMs are currently supported - more options could be added in the future
- Servers are deployed in the nbg1 datacenter - this could be made configurable in the future
- Servers are deployed using the docker-ce image - this could be made configurable in the future

