/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	v1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	health "github.com/argoproj/gitops-engine/pkg/health"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

const (
	applicationDeletionTimeout int = 120
)

// ApplicationCleanup removes and waits for specific ArgoCD applications
func ApplicationCleanup(clientset kubernetes.Interface, removeApps []string) error {
	// Patch registry app to remove syncPolicy
	removeSyncPolicyPatch, _ := json.Marshal(
		[]PatchStringValue{{
			Op:    "remove",
			Path:  "/spec/syncPolicy",
			Value: "",
		}})
	err := RestPatchArgoCD(clientset, "registry", removeSyncPolicyPatch)
	if err != nil {
		log.Error().Msgf("unable to patch registry application: %s", err)
		return fmt.Errorf("unable to patch registry application: %w", err)
	}

	log.Info().Msgf("removed syncPolicy from registry application or it was already disabled")

	// Patch dependent applications to remove syncPolicy}
	for _, app := range removeApps {
		log.Info().Msgf("attempting to delete argocd application %s", app)
		if err := waitForApplicationDeletion(clientset, app); err != nil {
			log.Error().Msgf("error deleting argocd application %q: %s", app, err)
			return fmt.Errorf("error deleting argocd application %q: %w", app, err)
		}
	}

	return nil
}

// deleteArgoCDApplicationV2 deletes an ArgoCD application
func deleteArgoCDApplicationV2(clientset kubernetes.Interface, applicationName string, ch chan<- bool) error {
	// Call the API to delete an ArgoCD application
	data, err := clientset.CoreV1().RESTClient().Delete().
		AbsPath(fmt.Sprintf("/apis/%s", ArgoCDAPIVersion)).
		Namespace("argocd").
		Resource("applications").
		Name(applicationName).
		DoRaw(context.Background())
	if err != nil {
		log.Error().Msgf("error deleting argocd application %q: %s", applicationName, err)
		return fmt.Errorf("error deleting argocd application %q: %w", applicationName, err)
	}

	// Unmarshal JSON API response to map[string]interface{}
	var resp map[string]interface{}
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Error().Msgf("unable to encode ArgoCD application data to JSON: %s", err.Error())
		return fmt.Errorf("unable to encode ArgoCD application data to JSON: %w", err)
	}

	log.Info().Msgf("deleting %s: %s", applicationName, strings.ToLower(fmt.Sprintf("%v", resp["status"])))

	for i := 0; i < applicationDeletionTimeout; i++ {
		status, _ := returnArgoCDApplicationStatus(clientset, applicationName)
		switch status {
		case health.HealthStatusUnknown:
			ch <- true
			return nil
		case health.HealthStatusMissing:
			ch <- true
			return nil
		case health.HealthStatusProgressing:
			log.Info().Msgf("application %s is progressing", applicationName)
			continue
		case health.HealthStatusDegraded:
			log.Info().Msgf("application %s is progressing", applicationName)
			continue
		}
		time.Sleep(time.Second * 1)
	}

	return nil
}

// RefreshRegistryApplication forces the registry application to fetch upstream manifests
func RefreshRegistryApplication(host string, token string) error {
	// Build request to ArgoCD API
	request, err := http.NewRequest(
		http.MethodGet,
		fmt.Sprintf("%s/api/v1/applications/registry?refresh=true", host),
		nil,
	)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Submit request to ArgoCD API
	client := &http.Client{Timeout: 10 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	return nil
}

// RefreshApplication forces the registry application to fetch upstream manifests
func RefreshApplication(host string, token string, appName string) error {
	// Build request to ArgoCD API
	endpoint := fmt.Sprintf("%s/api/v1/applications/%s?refresh=true", host, appName)
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("error creating request to refresh application %s: %w", appName, err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Submit request to ArgoCD API
	response, err := httpCommon.CustomHTTPClient(false).Do(request)
	if err != nil {
		return fmt.Errorf("error sending request to refresh application %s: %w", appName, err)
	}
	defer response.Body.Close()

	if _, err := io.Copy(io.Discard, response.Body); err != nil {
		return fmt.Errorf("error reading response body for application %s: %w", appName, err)
	}

	return nil
}

// returnArgoCDApplicationStatus returns the status details of a given ArgoCD application
func returnArgoCDApplicationStatus(clientset kubernetes.Interface, applicationName string) (health.HealthStatusCode, error) {
	// Call the API to return an ArgoCD application object
	data, err := clientset.CoreV1().RESTClient().Get().
		AbsPath(fmt.Sprintf("/apis/%s", ArgoCDAPIVersion)).
		Namespace("argocd").
		Resource("applications").
		Name(applicationName).
		DoRaw(context.Background())
	if err != nil {
		log.Error().Msgf("error retrieving argocd application %q: %s", applicationName, err)
		return health.HealthStatusUnknown, fmt.Errorf("error retrieving argocd application %q: %w", applicationName, err)
	}

	// Unmarshal JSON API response to map[string]interface{}
	var resp v1alpha1.Application
	if err := json.Unmarshal(data, &resp); err != nil {
		log.Error().Msgf("error converting argocd application data to v1alpha1.Application: %s", err)
		return health.HealthStatusUnknown, fmt.Errorf("error converting argocd application data to v1alpha1.Application: %w", err)
	}

	return resp.Status.Health.Status, nil
}

// waitForApplicationDeletion disables sync and deletes specific applications
// from ArgoCD before continuing
func waitForApplicationDeletion(clientset kubernetes.Interface, applicationName string) error {
	ch := make(chan bool)
	// Patch app to remove sync
	removeSyncPolicyPatch, _ := json.Marshal(
		[]PatchStringValue{{
			Op:    "remove",
			Path:  "/spec/syncPolicy",
			Value: "",
		}})
	err := RestPatchArgoCD(clientset, applicationName, removeSyncPolicyPatch)
	if err != nil {
		log.Info().Msgf("error patching argocd application %s: %s", applicationName, err)
		return fmt.Errorf("error patching argocd application %s: %w", applicationName, err)
	}

	log.Info().Msgf("removed syncPolicy from argocd application %s or it was not present", applicationName)

	// Delete applications and wait for them to report as deleted
	go deleteArgoCDApplicationV2(clientset, applicationName, ch)
	for {
		select {
		case deleted, ok := <-ch:
			if !ok || deleted {
				log.Info().Msgf("deleted argocd application %s if it existed", applicationName)
				return nil
			}
		case <-time.After(time.Duration(applicationDeletionTimeout) * time.Second):
			return fmt.Errorf("timed out waiting for argocd application %s to delete", applicationName)
		}
	}
}
