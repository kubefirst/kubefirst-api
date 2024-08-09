/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/rs/zerolog/log"
)

func NewDockerClient() *client.Client {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Error().Msgf("error instantiating docker client: %s", err)
		return nil
	}

	return cli
}

func (docker ClientWrapper) ListContainers() {
	containers, err := docker.Client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		log.Error().Msg(err.Error())
	}

	for _, container := range containers {
		fmt.Printf("%s %s\n", container.ID[:10], container.Image)
	}
}

// CheckDockerReady
func (docker ClientWrapper) CheckDockerReady() (bool, error) {
	_, err := docker.Client.Info(context.Background())
	if err != nil {
		log.Error().Msgf("error determining docker readiness: %s", err)
		return false, fmt.Errorf("error determining docker readiness: %s", err)
	}

	return true, nil
}
