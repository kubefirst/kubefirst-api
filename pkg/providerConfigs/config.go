/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs //nolint:revive // allowing temporarily for better code organization

import (
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

type ProviderConfig struct {
	AkamaiToken                      string
	CivoToken                        string
	DigitaloceanToken                string
	GoogleAuth                       string
	GoogleProject                    string
	K3sServersPrivateIps             []string
	K3sServersPublicIps              []string
	K3sSSHPrivateKey                 string
	K3sServersArgs                   []string
	K3sSSHUser                       string
	VultrToken                       string
	CloudflareAPIToken               string
	CloudflareOriginCaIssuerAPIToken string

	GithubToken string
	GitlabToken string

	ArgoWorkflowsDir                string
	DestinationGitopsRepoHTTPSURL   string
	DestinationGitopsRepoGitURL     string
	DestinationGitopsRepoURL        string
	DestinationMetaphorRepoHTTPSURL string
	DestinationMetaphorRepoGitURL   string
	DestinationMetaphorRepoURL      string
	GitopsDir                       string
	GitProvider                     string
	GitProtocol                     string
	K1Dir                           string
	Kubeconfig                      string
	KubectlClient                   string
	KubefirstBotSSHPrivateKey       string
	KubefirstConfig                 string
	LogsDir                         string
	MetaphorDir                     string
	RegistryAppName                 string
	RegistryYaml                    string
	SSLBackupDir                    string
	TerraformClient                 string
	ToolsDir                        string

	GitopsDirectoryValues   *GitopsDirectoryValues
	MetaphorDirectoryValues *MetaphorTokenValues
}

// GetConfig - load default values from kubefirst installer
func GetConfig(
	clusterName,
	domainName,
	gitProvider,
	gitOwner,
	gitProtocol,
	cloudflareAPIToken,
	cloudflareOriginCaIssuerAPIToken string,
) (*ProviderConfig, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("unable to get current user's home directory: %s", err)
		return nil, fmt.Errorf("unable to get current user's home directory: %w", err)
	}

	// cGitHost describes which git host to use depending on gitProvider
	var cGitHost string
	switch gitProvider {
	case "github":
		cGitHost = GithubHost
	case "gitlab":
		cGitHost = GitlabHost
	}

	return &ProviderConfig{
		DestinationGitopsRepoURL:         fmt.Sprintf("https://%s/%s/gitops.git", cGitHost, gitOwner),
		DestinationGitopsRepoGitURL:      fmt.Sprintf("git@%s:%s/gitops.git", cGitHost, gitOwner),
		DestinationMetaphorRepoURL:       fmt.Sprintf("https://%s/%s/metaphor.git", cGitHost, gitOwner),
		DestinationMetaphorRepoGitURL:    fmt.Sprintf("git@%s:%s/metaphor.git", cGitHost, gitOwner),
		ArgoWorkflowsDir:                 fmt.Sprintf("%s/.k1/%s/argo-workflows", homeDir, clusterName),
		GitopsDir:                        fmt.Sprintf("%s/.k1/%s/gitops", homeDir, clusterName),
		GitProvider:                      gitProvider,
		GitProtocol:                      gitProtocol,
		CloudflareAPIToken:               cloudflareAPIToken,
		CloudflareOriginCaIssuerAPIToken: cloudflareOriginCaIssuerAPIToken,
		Kubeconfig:                       fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName),
		K1Dir:                            fmt.Sprintf("%s/.k1/%s", homeDir, clusterName),
		KubectlClient:                    fmt.Sprintf("%s/.k1/%s/tools/kubectl", homeDir, clusterName),
		KubefirstConfig:                  fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, ".kubefirst"),
		LogsDir:                          fmt.Sprintf("%s/.k1/%s/logs", homeDir, clusterName),
		MetaphorDir:                      fmt.Sprintf("%s/.k1/%s/metaphor", homeDir, clusterName),
		RegistryAppName:                  "registry",
		RegistryYaml:                     fmt.Sprintf("%s/.k1/%s/gitops/registry/%s/registry.yaml", homeDir, clusterName, clusterName),
		SSLBackupDir:                     fmt.Sprintf("%s/.k1/%s/ssl/%s", homeDir, clusterName, domainName),
		TerraformClient:                  fmt.Sprintf("%s/.k1/%s/tools/terraform", homeDir, clusterName),
		ToolsDir:                         fmt.Sprintf("%s/.k1/%s/tools", homeDir, clusterName),
	}, nil
}
