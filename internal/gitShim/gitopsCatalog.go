/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim //nolint:revive // allowed during code reorg

import (
	"context"
	"errors"
	"io"

	"github.com/google/go-github/v52/github"
)

const (
	KubefirstGitHubOrganization      = "kubefirst"
	KubefirstGitopsCatalogRepository = "gitops-catalog"
	basePath                         = "/"
)

// GetGitopsCatalogRepo returns an object detailing the Kubefirst gitops catalog GitHub repository
func (gh *GitHubClient) GetGitopsCatalogRepo() (*github.Repository, error) {
	repo, _, err := gh.Client.Repositories.Get(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
	)
	if err != nil {
		return &github.Repository{}, err
	}

	return repo, nil
}

// ReadGitopsCatalogRepoContents reads the file and directory contents of the Kubefirst gitops catalog
// GitHub repository
func (gh *GitHubClient) ReadGitopsCatalogRepoContents() ([]*github.RepositoryContent, error) {
	_, directoryContent, _, err := gh.Client.Repositories.GetContents(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
		basePath,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return directoryContent, nil
}

// ReadGitopsCatalogRepoDirectory reads the files in a gitops catalog repo directory
func (gh *GitHubClient) ReadGitopsCatalogRepoDirectory(path string) ([]*github.RepositoryContent, error) {
	_, directoryContent, _, err := gh.Client.Repositories.GetContents(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
		path,
		nil,
	)
	if err != nil {
		return nil, err
	}

	return directoryContent, nil
}

// ReadGitopsCatalogIndex reads the gitops catalog repository index
func (gh *GitHubClient) ReadGitopsCatalogIndex(contents []*github.RepositoryContent) ([]byte, error) {
	for _, content := range contents {
		if *content.Type == "file" && *content.Name == "index.yaml" {
			b, err := gh.readFileContents(content)
			if err != nil {
				return b, err
			}
			return b, nil
		}
	}

	return []byte{}, errors.New("index.yaml not found in gitops catalog repository")
}

// ReadGitopsCatalogAppDirectory reads the file content in a gitops catalog app directory
func (gh *GitHubClient) ReadGitopsCatalogAppDirectory(contents []*github.RepositoryContent, applicationName string) ([][]byte, error) {
	for _, content := range contents {
		if *content.Type == "dir" && *content.Name == applicationName {
			files, err := gh.ReadGitopsCatalogRepoDirectory(*content.Path)
			if err != nil {
				return [][]byte{}, err
			}

			var res [][]byte

			for _, file := range files {
				b, err := gh.readFileContents(file)
				if err != nil {
					return [][]byte{}, err
				}
				res = append(res, b)
			}

			return res, nil
		}

		continue
	}

	return [][]byte{}, nil
}

// readFileContents parses the contents of a file in a GitHub repository
func (gh *GitHubClient) readFileContents(content *github.RepositoryContent) ([]byte, error) {
	rc, _, err := gh.Client.Repositories.DownloadContents(
		context.Background(),
		KubefirstGitHubOrganization,
		KubefirstGitopsCatalogRepository,
		*content.Path,
		nil,
	)
	if err != nil {
		return []byte{}, err
	}
	defer rc.Close()

	b, err := io.ReadAll(rc)
	if err != nil {
		return []byte{}, err
	}

	return b, nil
}
