/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package github

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
)

const (
	githubAPIURL = "https://api.github.com"
)

var requiredScopes = [...]string{
	"admin:org",
	"admin:public_key",
	"admin:repo_hook",
	"delete_repo",
	"repo",
	"user",
	"workflow",
	"write:packages",
}

// VerifyTokenPermissions compares scope of the provided token to the required
// scopes for kubefirst functionality
func VerifyTokenPermissions(githubToken string) error {
	req, err := http.NewRequest(http.MethodGet, githubAPIURL, nil)
	if err != nil {
		log.Info().Msg("error setting github owner permissions request")
		return fmt.Errorf("unable to create request to verify token permissions: %w", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", githubToken))

	res, err := httpCommon.CustomHTTPClient(false).Do(req)
	if err != nil {
		return fmt.Errorf("error calling GitHub API %q: %s", req.URL.String(), err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading GitHub's response body for %q: %w", req.URL.String(), err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"something went wrong calling GitHub API, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	// Get token scopes
	scopeHeader := res.Header.Get("X-OAuth-Scopes")
	scopes := make([]string, 0)
	for _, s := range strings.Split(scopeHeader, ",") {
		scopes = append(scopes, strings.TrimSpace(s))
	}

	// Compare token scopes to required scopes
	missingScopes := make([]string, 0)
	for _, ts := range requiredScopes {
		if !pkg.FindStringInSlice(scopes, ts) {
			missingScopes = append(missingScopes, ts)
		}
	}

	// Report on any missing scopes
	if len(missingScopes) != 0 {
		return fmt.Errorf("the supplied github token is missing authorization scopes - please add: %v", missingScopes)
	}

	return nil
}
