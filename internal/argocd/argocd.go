/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	v1alpha1ArgocdApplication "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocdModel"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Types "k8s.io/client-go/kubernetes/typed/core/v1"
)

var ArgocdSecretClient coreV1Types.SecretInterface

// todo call this ArgocdConfig or something not so generic
// Config ArgoCD configuration
type Config struct {
	Configs struct {
		Repositories struct {
			SoftServeGitops struct {
				URL      string `yaml:"url,omitempty"`
				Insecure string `json:"insecure,omitempty"`
				Type     string `json:"type,omitempty"`
				Name     string `json:"name,omitempty"`
			} `yaml:"soft-serve-gitops,omitempty"`
			RepoGitops struct {
				URL  string `yaml:"url,omitempty"`
				Type string `yaml:"type,omitempty"`
				Name string `yaml:"name,omitempty"`
			} `yaml:"github-serve-gitops,omitempty"`
		} `yaml:"repositories,omitempty"`
		CredentialTemplates struct {
			SSHCreds struct {
				URL           string `yaml:"url,omitempty"`
				SSHPrivateKey string `yaml:"sshPrivateKey,omitempty"`
			} `yaml:"ssh-creds,omitempty"`
		} `yaml:"credentialTemplates,omitempty"`
	} `yaml:"configs,omitempty"`
	Server struct {
		ExtraArgs []string `yaml:"extraArgs,omitempty"`
		Ingress   struct {
			Enabled     string `yaml:"enabled,omitempty"`
			Annotations struct {
				IngressKubernetesIoRewriteTarget   string `yaml:"ingress.kubernetes.io/rewrite-target,omitempty"`
				IngressKubernetesIoBackendProtocol string `yaml:"ingress.kubernetes.io/backend-protocol,omitempty"`
			} `yaml:"annotations,omitempty"`
			Hosts []string    `yaml:"hosts,omitempty"`
			TLS   []TLSConfig `yaml:"tls,omitempty"`
		} `yaml:"ingress,omitempty"`
	} `yaml:"server,omitempty"`
}

type TLSConfig struct {
	Hosts      []string `yaml:"hosts,omitempty"`
	SecretName string   `yaml:"secretName,omitempty"`
}

// Sync request ArgoCD to manual sync an application.
func DeleteApplication(httpClient pkg.HTTPDoer, applicationName, argoCDToken, cascade string) (int, string, error) {
	params := url.Values{}
	params.Add("cascade", cascade)
	paramBody := strings.NewReader(params.Encode())

	url := fmt.Sprintf("%s/api/v1/applications/%s", GetArgoEndpoint(), applicationName)
	log.Info().Msgf("deleting application %s using endpoint %q", applicationName, url)

	req, err := http.NewRequest(http.MethodDelete, url, paramBody)
	if err != nil {
		log.Error().Msgf("error creating DELETE request to ArgoCD for application %q: %s", applicationName, err.Error())
		return 0, "", fmt.Errorf("error creating DELETE request to ArgoCD for application %q: %w", applicationName, err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", argoCDToken))

	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Msgf("error sending DELETE request to ArgoCD for application %q: %s", applicationName, err.Error())
		return res.StatusCode, "", fmt.Errorf("error sending DELETE request to ArgoCD for application %q: %w", applicationName, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		log.Warn().Err(err).Msgf("argocd http response code for delete action is: %d", res.StatusCode)
		return res.StatusCode, "", nil
	}

	var syncResponse argocdModel.V1alpha1Application
	if err := json.NewDecoder(res.Body).Decode(&syncResponse); err != nil {
		log.Error().Msgf("error decoding response body for application %q: %s", applicationName, err.Error())
		return res.StatusCode, "", fmt.Errorf("error decoding response body for application %q: %w", applicationName, err)
	}

	return res.StatusCode, syncResponse.Status.Sync.Status, nil
}

// GetArgoCDApplication by receiving the ArgoCD token, and the application name, this function returns the full
// application data Application struct. This can be used when a resource needs to be updated, we firstly collect all
// Application data, update what is necessary and then request the PUT function to update the resource.
func GetArgoCDApplication(token string, applicationName string) (*argocdModel.V1alpha1Application, error) {
	httpClient := httpCommon.CustomHTTPClient(true)

	url := pkg.ArgoCDLocalBaseURL + "/applications/" + applicationName
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Error().Msgf("error creating GET request to ArgoCD for application %q: %s", applicationName, err.Error())
		return nil, fmt.Errorf("error creating GET request to ArgoCD for application %q: %w", applicationName, err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := httpClient.Do(req)
	if err != nil {
		log.Error().Msgf("error sending GET request to ArgoCD for application %q: %s", applicationName, err.Error())
		return nil, fmt.Errorf("error sending GET request to ArgoCD for application %q: %w", applicationName, err)
	}
	defer res.Body.Close()

	var response argocdModel.V1alpha1Application
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		log.Error().Msgf("error decoding response body for application %q: %s", applicationName, err.Error())
		return nil, fmt.Errorf("error decoding response body for application %q: %w", applicationName, err)
	}

	return &response, nil
}

// GetArgoEndpoint provides a solution in the interim for returning the correct
// endpoint address
func GetArgoEndpoint() string {
	var argoCDLocalEndpoint string
	if viper.GetString("argocd.local.service") != "" {
		argoCDLocalEndpoint = viper.GetString("argocd.local.service")
	} else {
		argoCDLocalEndpoint = pkg.ArgocdPortForwardURL
	}
	return argoCDLocalEndpoint
}

func GetArgoCDApplicationObject(gitopsRepoURL, registryPath string) *v1alpha1ArgocdApplication.Application {
	return &v1alpha1ArgocdApplication.Application{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Application",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "registry",
			Namespace:   "argocd",
			Annotations: map[string]string{"argocd.argoproj.io/sync-wave": "1"},
		},
		Spec: v1alpha1ArgocdApplication.ApplicationSpec{
			Source: &v1alpha1ArgocdApplication.ApplicationSource{
				RepoURL:        gitopsRepoURL,
				Path:           registryPath,
				TargetRevision: "HEAD",
			},
			Destination: v1alpha1ArgocdApplication.ApplicationDestination{
				Server:    "https://kubernetes.default.svc",
				Namespace: "argocd",
			},
			Project: "default",
			SyncPolicy: &v1alpha1ArgocdApplication.SyncPolicy{
				Automated: &v1alpha1ArgocdApplication.SyncPolicyAutomated{
					Prune:    true,
					SelfHeal: true,
				},
				SyncOptions: []string{"CreateNamespace=true"},
				Retry: &v1alpha1ArgocdApplication.RetryStrategy{
					Limit: 5,
					Backoff: &v1alpha1ArgocdApplication.Backoff{
						Duration:    "5s",
						Factor:      new(int64),
						MaxDuration: "5m0s",
					},
				},
			},
		},
	}
}
