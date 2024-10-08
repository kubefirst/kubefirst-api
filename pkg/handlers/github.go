/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"

	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/httpCommon"
	"github.com/konstructio/kubefirst-api/internal/services"
	"github.com/konstructio/kubefirst-api/pkg/reports"
)

// GitHubDeviceFlow handles https://docs.github.com/apps/building-oauth-apps/authorizing-oauth-apps#device-flow
type GitHubDeviceFlow struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

type GitHubUser struct {
	Login string `json:"login"`
}

type GitHubOrganizationRole struct {
	Role string `json:"role"`
}

// GitHubHandler receives a GitHubService
type GitHubHandler struct {
	service *services.GitHubService
}

// NewGitHubHandler instantiate a new GitHub handler
func NewGitHubHandler(gitHubService *services.GitHubService) *GitHubHandler {
	return &GitHubHandler{
		service: gitHubService,
	}
}

// AuthenticateUser initiate the GitHub Device Login Flow. First step is to issue a new device, and user code. Next it
// waits for the user authorize the request in the browser, then it pool GitHub access point endpoint, to validate and
// grant permission to return a valid access token.
func (handler GitHubHandler) AuthenticateUser() (string, error) {
	gitHubDeviceFlowCodeURL := "https://github.com/login/device/code"
	// todo: update scope list, we have more than we need at the moment
	requestBody, err := json.Marshal(map[string]string{
		"client_id": pkg.GitHubOAuthClientID,
		"scope":     "repo public_repo admin:repo_hook admin:org admin:public_key admin:org_hook user project delete_repo write:packages admin:gpg_key workflow",
	})
	if err != nil {
		return "", fmt.Errorf("error marshalling request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, gitHubDeviceFlowCodeURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", fmt.Errorf("error setting request: %w", err)
	}
	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", pkg.JSONContentType)

	res, err := httpCommon.CustomHTTPClient(false).Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling GitHub API: %w", err)
	}
	defer res.Body.Close()

	var gitHubDeviceFlow GitHubDeviceFlow
	if err := json.NewDecoder(res.Body).Decode(&gitHubDeviceFlow); err != nil {
		return "", fmt.Errorf("error decoding response body: %w", err)
	}

	// todo: check http code

	// UI update to the user adding instructions how to proceed
	gitHubTokenReport := printGitHubAuthToken(gitHubDeviceFlow.UserCode, gitHubDeviceFlow.VerificationURI)
	fmt.Println(reports.StyleMessage(gitHubTokenReport))

	// // this blocks the progress until the user hits enter to open the browser
	// if _, err = fmt.Scanln(); err != nil {
	// 	return "", err
	// }

	if err = pkg.OpenBrowser("https://github.com/login/device"); err != nil {
		log.Error().Msgf("error opening browser: %s", err)
		return "", fmt.Errorf("error opening browser: %w", err)
	}

	var gitHubAccessToken string
	attempts := 18       // 18 * 5 = 90 seconds
	secondsControl := 95 // 95 to start with 95-5=90
	for i := 0; i < attempts; i++ {
		gitHubAccessToken, err = handler.service.CheckUserCodeConfirmation(gitHubDeviceFlow.DeviceCode)
		if err != nil {
			log.Warn().Msgf("%s", err)
		}

		if len(gitHubAccessToken) > 0 {
			fmt.Printf("\n\nGitHub access token set!\n\n")
			return gitHubAccessToken, nil
		}

		secondsControl -= 5
		fmt.Printf("\rwaiting for authorization (%d seconds)", secondsControl)
		// todo: handle github interval https://docs.github.com/en/developers/apps/building-oauth-apps/authorizing-oauth-apps#response-parameters
		time.Sleep(5 * time.Second)
	}
	fmt.Println("") // will avoid writing the next print in the same line
	return gitHubAccessToken, nil
}

// todo: make it a method
func (handler GitHubHandler) GetGitHubUser(gitHubAccessToken string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		log.Warn().Msg("error setting request")
		return "", fmt.Errorf("error setting request: %w", err)
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", gitHubAccessToken))

	res, err := httpCommon.CustomHTTPClient(false).Do(req)
	if err != nil {
		return "", fmt.Errorf("error calling GitHub API: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf(
			"something went wrong calling GitHub API during user lookup, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	var githubUser GitHubUser
	if err := json.Unmarshal(body, &githubUser); err != nil {
		return "", fmt.Errorf("error unmarshalling response body: %w", err)
	}

	if len(githubUser.Login) == 0 {
		return "", errors.New("unable to retrieve username via GitHub API")
	}

	log.Info().Msgf("GitHub user: %s", githubUser.Login)
	return githubUser.Login, nil
}

func (handler GitHubHandler) CheckGithubOrganizationPermissions(githubToken, githubOwner, githubUsername string) error {
	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/orgs/%s/memberships/%s", githubOwner, githubUsername), nil)
	if err != nil {
		log.Info().Msg("error setting github owner permissions request")
		return fmt.Errorf("error setting github owner permissions request: %w", err)
	}

	req.Header.Add("Content-Type", pkg.JSONContentType)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", githubToken))

	res, err := httpCommon.CustomHTTPClient(false).Do(req)
	if err != nil {
		return fmt.Errorf("error calling GitHub API: %w", err)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"something went wrong calling GitHub API during org lookup, http status code is: %d, and response is: %q",
			res.StatusCode,
			string(body),
		)
	}

	var gitHubOrganizationRole GitHubOrganizationRole
	err = json.Unmarshal(body, &gitHubOrganizationRole)
	if err != nil {
		return fmt.Errorf("error unmarshalling response body: %w", err)
	}

	log.Info().Msgf("the github owner role is: %s", gitHubOrganizationRole.Role)

	if gitHubOrganizationRole.Role != "admin" {
		errMsg := fmt.Sprintf("Authenticated user (via GITHUB_TOKEN) doesn't have adequate permissions.\n Make sure they are an `Owner` in %s.\n Current role: %s", githubOwner, gitHubOrganizationRole.Role)
		return errors.New(errMsg)
	}

	return nil
}

func printGitHubAuthToken(userCode, verificationURI string) string {
	var gitHubTokenReport bytes.Buffer
	gitHubTokenReport.WriteString(strings.Repeat("-", 69))
	gitHubTokenReport.WriteString("\nNo GITHUB_TOKEN env variable found!\nUse the code below to get a temporary GitHub Access Token\nThis token will be used by Kubefirst to create your environment\n")
	gitHubTokenReport.WriteString("\n\nA GitHub Access Token is required to provision GitHub repositories and run workflows in GitHub.\n")
	gitHubTokenReport.WriteString(strings.Repeat("-", 69) + "\n")
	gitHubTokenReport.WriteString("1. Copy this code: 📋 " + userCode + " 📋\n\n")
	gitHubTokenReport.WriteString(fmt.Sprintf("2. When ready, press <enter> to open the page at %s\n\n", verificationURI))
	gitHubTokenReport.WriteString("3. Authorize the organization you'll be using Kubefirst with - this may also be your personal account")

	return gitHubTokenReport.String()
}
