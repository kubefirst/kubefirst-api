/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitShim //nolint:revive // allowing name during code cleanup

import (
	"github.com/google/go-github/v52/github"
)

// GitHubClient acts as a receiver for interacting with GitHub's API
type GitHubClient struct {
	Client *github.Client
}

// NewGitHub instantiates an unauthenticated GitHub client
func NewGitHub() *github.Client {
	return github.NewClient(nil)
}
