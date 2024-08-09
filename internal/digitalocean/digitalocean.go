/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
)

// GetRegions lists all available regions
func (c *Configuration) GetRegions() ([]string, error) {
	regions, _, err := c.Client.Regions.List(c.Context, &godo.ListOptions{})
	if err != nil {
		return nil, err
	}
	regionList := make([]string, 0, len(regions))
	for _, region := range regions {
		regionList = append(regionList, region.Slug)
	}

	return regionList, nil
}

func (c *Configuration) ListInstances() ([]string, error) {
	maxItemsPerPage := 200
	instances, _, err := c.Client.Sizes.List(context.Background(), &godo.ListOptions{PerPage: maxItemsPerPage})
	if err != nil {
		return nil, err
	}

	instanceNames := make([]string, 0, len(instances))
	for _, instance := range instances {
		instanceNames = append(instanceNames, instance.Slug)
	}

	return instanceNames, nil
}

func (c *Configuration) GetKubeconfig(clusterName string) ([]byte, error) {
	clusters, _, err := c.Client.Kubernetes.List(context.Background(), &godo.ListOptions{})
	if err != nil {
		return nil, err
	}

	var clusterID string
	for _, cluster := range clusters {
		if cluster.Name == clusterName {
			clusterID = cluster.ID
			continue
		}
	}

	if clusterID == "" {
		return nil, fmt.Errorf("could not find cluster ID for cluster name %s", clusterName)
	}

	config, _, err := c.Client.Kubernetes.GetKubeConfig(context.Background(), clusterID)
	if err != nil {
		return nil, err
	}

	return config.KubeconfigYAML, nil
}
