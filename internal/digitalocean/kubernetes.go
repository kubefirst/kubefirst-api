/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"errors"
	"fmt"
	"time"

	"github.com/digitalocean/godo"
	"github.com/rs/zerolog/log"
)

// GetKubernetesAssociatedResources returns resources associated with a digitalocean Kubernetes cluster
func (c *Configuration) GetKubernetesAssociatedResources(clusterName string) (*godo.KubernetesAssociatedResources, error) {
	clusters, _, err := c.Client.Kubernetes.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error listing kubernetes clusters: %w", err)
	}

	var clusterID string
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			clusterID = cluster.ID
		}
	}
	if clusterID == "" {
		return nil, fmt.Errorf("could not find cluster ID for cluster name %w", err)
	}

	resources, _, err := c.Client.Kubernetes.ListAssociatedResourcesForDeletion(c.Context, clusterID)
	if err != nil {
		return nil, fmt.Errorf("error listing associated resources: %w", err)
	}

	return resources, nil
}

// DeleteKubernetesClusterVolumes iterates over resource volumes and deletes them
func (c *Configuration) DeleteKubernetesClusterVolumes(resources *godo.KubernetesAssociatedResources) error {
	if len(resources.Volumes) == 0 {
		return errors.New("no volumes are available for deletion with the provided parameters")
	}

	for _, vol := range resources.Volumes {
		// Wait for volume to unattach
		for i := 0; i < 24; i++ {
			voldata, _, err := c.Client.Storage.GetVolume(c.Context, vol.ID)
			if err != nil {
				return fmt.Errorf("error getting volume %s: %w", vol.ID, err)
			}
			if len(voldata.DropletIDs) != 0 {
				log.Info().Msgf("volume %s is still attached to droplet(s) - waiting...", vol.ID)
			} else {
				break
			}
			time.Sleep(time.Second * 5)
		}

		log.Info().Msg("removing volume with name: " + vol.Name)
		_, err := c.Client.Storage.DeleteVolume(c.Context, vol.ID)
		if err != nil {
			log.Error().Msgf("error deleting volume %s: %s", vol.ID, err)
			return fmt.Errorf("error deleting volume %s: %w", vol.ID, err)
		}
		log.Info().Msg("volume " + vol.ID + " deleted")
	}

	return nil
}
