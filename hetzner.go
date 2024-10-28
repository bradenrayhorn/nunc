package main

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type hetzner struct {
	client *hcloud.Client
	key    sshKey
}

const cloudInit = `#cloud-config
users:
  - name: gh
    groups: docker
    sudo: ALL=(ALL) NOPASSWD:ALL
    ssh_authorized_keys:
      - "%s"
`

func newHetzner(token string) (*hetzner, error) {
	client := hcloud.NewClient(hcloud.WithToken(token))

	key, err := newSSHKey()
	if err != nil {
		return nil, err
	}

	return &hetzner{
		client: client,
		key:    key,
	}, nil
}

func (h *hetzner) listServers() ([]*hcloud.Server, error) {
	return h.client.Server.All(context.Background())
}

func (h *hetzner) deleteServer(server *hcloud.Server) error {
	_, _, err := h.client.Server.DeleteWithResult(context.Background(), server)
	return err
}

func (h *hetzner) newServer(
	ctx context.Context,
	size string,
	jobID string,
) (*hcloud.Server, error) {
	startAfterCreate := true

	result, _, err := h.client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:             jobID,
		ServerType:       &hcloud.ServerType{Name: size},
		Image:            &hcloud.Image{Name: "docker-ce"},
		Location:         &hcloud.Location{Name: "nbg1"},
		PublicNet:        &hcloud.ServerCreatePublicNet{EnableIPv4: true},
		StartAfterCreate: &startAfterCreate,
		UserData:         fmt.Sprintf(cloudInit, h.key.pubKeyString),
	})
	if err != nil {
		return nil, fmt.Errorf("create hetzner server: %w", err)
	}

	return result.Server, nil
}

func (h *hetzner) reimageServer(
	ctx context.Context,
	server *hcloud.Server,
) error {
	_, _, err := h.client.Server.RebuildWithResult(ctx, server, hcloud.ServerRebuildOpts{
		Image: &hcloud.Image{Name: "docker-ce"},
	})
	if err != nil {
		return fmt.Errorf("rebuild hetzner server: %w", err)
	}

	return nil
}
