/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitopsCatalog //nolint:revive // temporary allowing during code organization

import (
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"gopkg.in/yaml.v2"
)

// ReadActiveApplications reads the active upstream application manifest
func ReadActiveApplications() (types.GitopsCatalogApps, error) {
	gh := gitShim.GitHubClient{
		Client: gitShim.NewGitHub(),
	}

	activeContent, err := gh.ReadGitopsCatalogRepoContents()
	if err != nil {
		return types.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog repository content: %w", err)
	}

	index, err := gh.ReadGitopsCatalogIndex(activeContent)
	if err != nil {
		return types.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog index content: %w", err)
	}

	var out types.GitopsCatalogApps

	err = yaml.Unmarshal(index, &out)
	if err != nil {
		return types.GitopsCatalogApps{}, fmt.Errorf("error retrieving gitops catalog applications: %w", err)
	}

	return out, nil
}

// ReadApplicationDirectory reads a gitops catalog application's directory
func ReadApplicationDirectory(applicationName string) ([][]byte, error) {
	gh := gitShim.GitHubClient{
		Client: gitShim.NewGitHub(),
	}

	activeContent, err := gh.ReadGitopsCatalogRepoContents()
	if err != nil {
		return [][]byte{}, fmt.Errorf("error retrieving gitops catalog app directory content: %w", err)
	}

	contents, err := gh.ReadGitopsCatalogAppDirectory(activeContent, applicationName)
	if err != nil {
		return [][]byte{}, fmt.Errorf("error retrieving gitops catalog app directory content: %w", err)
	}

	return contents, nil
}
