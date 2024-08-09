/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocdModel"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
)

func getToken(endpoint, username, password string) (string, error) {
	httpClient := httpCommon.CustomHTTPClient(true)
	argoCDConfig := argocdModel.SessionSessionCreateRequest{
		Username: username,
		Password: password,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(argoCDConfig); err != nil {
		return "", fmt.Errorf("unable to encode argocd config to JSON: %w", err)
	}

	res, err := httpClient.Post(endpoint, "application/json", &buf)
	if err != nil {
		return "", fmt.Errorf("unable to retrieve argocd token: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unable to retrieve argocd token: status code was %d", res.StatusCode)
	}

	var response struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("unable to decode argocd token response: %w", err)
	}

	return response.Token, nil
}

// GetArgoCDToken expects ArgoCD username and password, and returns a ArgoCD Bearer Token. ArgoCD username and password
// are stored in the viper file.
func GetArgoCDToken(username string, password string) (string, error) {
	url := pkg.ArgoCDLocalBaseURL + "/session"
	return getToken(url, username, password)
}

// GetArgocdTokenV2
func GetArgocdTokenV2(argocdBaseURL string, username string, password string) (string, error) {
	url := argocdBaseURL + "/api/v1/session"
	return getToken(url, username, password)
}
