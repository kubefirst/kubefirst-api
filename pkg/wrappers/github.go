/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package wrappers

import (
	"errors"

	"github.com/kubefirst/kubefirst-api/pkg/handlers"
)

// AuthenticateGitHubUserWrapper receives a handler that was previously instantiated, and communicate with GitHub.
// This wrapper is necessary to avoid code repetition when requesting GitHub PAT or Access token.
func AuthenticateGitHubUserWrapper(gitHubAccessToken string, gitHubHandler *handlers.GitHubHandler) (string, error) {
	if gitHubAccessToken != "" {
		return gitHubAccessToken, nil
	}

	gitHubAccessToken, err := gitHubHandler.AuthenticateUser()
	if err != nil {
		return "", err
	}

	if gitHubAccessToken == "" {
		return "", errors.New("unable to retrieve a GitHub token for the user")
	}

	return gitHubAccessToken, nil
}
