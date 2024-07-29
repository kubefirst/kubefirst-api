/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

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
	K3sSshPrivateKey                 string
	K3sServersArgs                   []string
	K3sSshUser                       string
	VultrToken                       string
	CloudflareAPIToken               string
	CloudflareOriginCaIssuerAPIToken string

	GithubToken string
	GitlabToken string

	ArgoWorkflowsDir                string
	DestinationGitopsRepoHttpsURL   string
	DestinationGitopsRepoGitURL     string
	DestinationGitopsRepoURL        string
	DestinationMetaphorRepoHttpsURL string
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
	clusterName string,
	domainName string,
	gitProvider string,
	gitOwner string,
	gitProtocol string,
	cloudflareAPIToken string,
	cloudflareOriginCaIssuerAPIToken string,
	GitopsRepoName string,
	MetaphorRepoName string,
) *ProviderConfig {

	config := ProviderConfig{}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("something went wrong getting home path: %s", err)
	}

	// cGitHost describes which git host to use depending on gitProvider
	var cGitHost string
	switch gitProvider {
	case "github":
		cGitHost = GithubHost
	case "gitlab":
		cGitHost = GitlabHost
	}

	config.DestinationGitopsRepoURL = fmt.Sprintf("https://%s/%s/%s.git", cGitHost, gitOwner,GitopsRepoName)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@%s:%s/%s.git", cGitHost, gitOwner,GitopsRepoName)
	config.DestinationMetaphorRepoURL = fmt.Sprintf("https://%s/%s/%s.git", cGitHost, gitOwner,MetaphorRepoName)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@%s:%s/%s.git", cGitHost, gitOwner,MetaphorRepoName)
	config.ArgoWorkflowsDir = fmt.Sprintf("%s/.k1/%s/argo-workflows", homeDir, clusterName)
	config.GitopsDir = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName,GitopsRepoName)
	config.GitProvider = gitProvider
	config.GitProtocol = gitProtocol
	config.CloudflareAPIToken = cloudflareAPIToken
	config.CloudflareOriginCaIssuerAPIToken = cloudflareOriginCaIssuerAPIToken
	config.Kubeconfig = fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName)
	config.K1Dir = fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
	config.KubectlClient = fmt.Sprintf("%s/.k1/%s/tools/kubectl", homeDir, clusterName)
	config.KubefirstConfig = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, ".kubefirst")
	config.LogsDir = fmt.Sprintf("%s/.k1/%s/logs", homeDir, clusterName)
	config.MetaphorDir = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, MetaphorRepoName)
	config.RegistryAppName = "registry"
	config.RegistryYaml = fmt.Sprintf("%s/.k1/%s/%s/registry/%s/registry.yaml", homeDir, clusterName, clusterName,GitopsRepoName)
	config.SSLBackupDir = fmt.Sprintf("%s/.k1/%s/ssl/%s", homeDir, clusterName, domainName)
	config.TerraformClient = fmt.Sprintf("%s/.k1/%s/tools/terraform", homeDir, clusterName)
	config.ToolsDir = fmt.Sprintf("%s/.k1/%s/tools", homeDir, clusterName)

	return &config
}
