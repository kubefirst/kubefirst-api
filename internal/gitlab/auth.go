/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitlab

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
)

const (
	gitlabAPIURL = "https://gitlab.com/api/v4"
)

// VerifyTokenPermissions compares scope of the provided token to the required
// scopes for kubefirst functionality
func VerifyTokenPermissions(gitlabToken string) error {
	destination := fmt.Sprintf("%s/personal_access_tokens/self", gitlabAPIURL)
	req, err := http.NewRequest(http.MethodGet, destination, nil)
	if err != nil {
		log.Error().Msgf("unable to create HTTP request to %q", destination)
		return fmt.Errorf("unable to create HTTP request to %q", destination)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gitlabToken))

	res, err := httpCommon.CustomHTTPClient(false).Do(req)
	if err != nil {
		return fmt.Errorf("unable to make GET request to GitLab API %q: %w", req.URL.String(), err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"something went wrong calling GitLab API, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	// Get token scopes
	var response struct {
		Scopes []string `json:"scopes"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return err
	}

	// api allows all access so we won't need to check the rest
	if internal.FindStringInSlice(response.Scopes, "api") {
		return nil
	}

	requiredScopes := [...]string{
		"read_api",
		"read_user",
		"read_repository",
		"write_repository",
		"read_registry",
		"write_registry",
	}

	// Compare token scopes to required scopes
	missingScopes := make([]string, 0)
	for _, ts := range requiredScopes {
		if !internal.FindStringInSlice(response.Scopes, ts) {
			missingScopes = append(missingScopes, ts)
		}
	}

	// Report on any missing scopes
	if !internal.FindStringInSlice(response.Scopes, "api") && len(missingScopes) != 0 {
		return fmt.Errorf("the supplied github token is missing authorization scopes - please add: %v", missingScopes)
	}

	return nil
}
