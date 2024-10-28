package main

import (
	"sync"
	"time"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

const (
	archARM64 = "arm64"
	archAMD64 = "amd64"
)

const (
	serverStatusWorking = "working"
	serverStatusIdle    = "idle"
	serverStatusDead    = "dead"
)

type serverState struct {
	mutex sync.Mutex

	runners map[string]runner
}

type runner struct {
	name string
	arch string

	server *hcloud.Server
	status string

	lastJobBeganAt time.Time
	createdAt      time.Time
}
