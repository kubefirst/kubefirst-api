/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"fmt"
	"os"
	"runtime"

	"github.com/caarlos0/env/v6"
	"github.com/rs/zerolog/log"
)

const (
	ArgocdPortForwardURL = "http://localhost:8080"
	CloudProvider        = "k3d"
	DomainName           = "kubefirst.dev"
	GithubHost           = "github.com"
	GitlabHost           = "gitlab.com"
	K3dVersion           = "v5.4.6"
	KubectlVersion       = "v1.25.7"
	LocalhostARCH        = runtime.GOARCH
	LocalhostOS          = runtime.GOOS
	MkCertVersion        = "v1.4.4"
	TerraformVersion     = "1.3.8"
	VaultPortForwardURL  = "http://localhost:8200"
)

var (
	ArgocdURL              = fmt.Sprintf("https://argocd.%s", DomainName)
	ArgoWorkflowsURL       = fmt.Sprintf("https://argo.%s", DomainName)
	AtlantisURL            = fmt.Sprintf("https://atlantis.%s", DomainName)
	ChartMuseumURL         = fmt.Sprintf("https://chartmuseum.%s", DomainName)
	KubefirstConsoleURL    = fmt.Sprintf("https://kubefirst.%s", DomainName)
	MetaphorDevelopmentURL = fmt.Sprintf("https://metaphor-devlopment.%s", DomainName)
	MetaphorStagingURL     = fmt.Sprintf("https://metaphor-staging.%s", DomainName)
	MetaphorProductionURL  = fmt.Sprintf("https://metaphor-production.%s", DomainName)
	VaultURL               = fmt.Sprintf("https://vault.%s", DomainName)
)

type K3dConfig struct {
	GithubToken string
	GitlabToken string

	DestinationGitopsRepoGitURL     string
	DestinationGitopsRepoURL        string
	DestinationMetaphorRepoURL      string
	DestinationMetaphorRepoGitURL   string
	DestinationGitopsRepoHttpsURL   string
	DestinationMetaphorRepoHttpsURL string
	GitopsDir                       string
	GitProvider                     string
	GitProtocol                     string
	K1Dir                           string
	K3dClient                       string
	Kubeconfig                      string
	KubectlClient                   string
	KubefirstConfig                 string
	MetaphorDir                     string
	MkCertClient                    string
	MkCertPemDir                    string
	MkCertSSLSecretDir              string
	TerraformClient                 string
	ToolsDir                        string
	GitopsRepoName					string
	MetaphorRepoName				string
	AdminTeamName					string
	DeveloperTeamName				string
}

// GetConfig - load default values from kubefirst installer
func GetConfig(clusterName string, gitProvider string, gitOwner string, gitProtocol string,gitopsRepoName string,metaphorRepoName string,adminTeamName string,developerTeamName string) *K3dConfig {
	config := K3dConfig{}

	if err := env.Parse(&config); err != nil {
		log.Error().Msgf("something went wrong loading the environment variables: %s", err)
	}

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

	config.DestinationGitopsRepoURL = fmt.Sprintf("https://%s/%s/%s.git", cGitHost, gitOwner, gitopsRepoName)
	config.DestinationGitopsRepoGitURL = fmt.Sprintf("git@%s:%s/%s.git", cGitHost, gitOwner, gitopsRepoName)
	config.DestinationMetaphorRepoURL = fmt.Sprintf("https://%s/%s/%s.git", cGitHost, gitOwner, metaphorRepoName)
	config.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@%s:%s/%s.git", cGitHost, gitOwner, metaphorRepoName)
	config.GitopsRepoName = gitopsRepoName
	config.MetaphorRepoName = metaphorRepoName
	config.AdminTeamName = adminTeamName
	config.DeveloperTeamName = developerTeamName
	config.GitopsDir = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, gitopsRepoName)
	config.GitProvider = gitProvider
	config.GitProtocol = gitProtocol
	config.K1Dir = fmt.Sprintf("%s/.k1/%s", homeDir, clusterName)
	config.K3dClient = fmt.Sprintf("%s/.k1/%s/tools/k3d", homeDir, clusterName)
	config.KubectlClient = fmt.Sprintf("%s/.k1/%s/tools/kubectl", homeDir, clusterName)
	config.Kubeconfig = fmt.Sprintf("%s/.k1/%s/kubeconfig", homeDir, clusterName)
	config.KubefirstConfig = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, ".kubefirst")
	config.MetaphorDir = fmt.Sprintf("%s/.k1/%s/%s", homeDir, clusterName, metaphorRepoName)
	config.MkCertClient = fmt.Sprintf("%s/.k1/%s/tools/mkcert", homeDir, clusterName)
	config.MkCertPemDir = fmt.Sprintf("%s/.k1/%s/ssl/%s/pem", homeDir, clusterName, DomainName)
	config.MkCertSSLSecretDir = fmt.Sprintf("%s/.k1/%s/ssl/%s/secrets", homeDir, clusterName, DomainName)
	config.TerraformClient = fmt.Sprintf("%s/.k1/%s/tools/terraform", homeDir, clusterName)
	config.ToolsDir = fmt.Sprintf("%s/.k1/%s/tools", homeDir, clusterName)

	return &config
}



type GitopsDirectoryValues struct {
	GithubOwner                   string
	GithubUser                    string
	GitlabOwner                   string
	GitlabOwnerGroupID            int
	GitlabUser                    string
	GitopsRepoGitURL              string
	GitopsRepoHttpsURL            string
	DomainName                    string
	AtlantisAllowList             string
	AlertsEmail                   string
	ClusterName                   string
	ClusterType                   string
	GithubHost                    string
	GitlabHost                    string
	ArgoWorkflowsIngressURL       string
	VaultIngressURL               string
	ArgocdIngressURL              string
	AtlantisIngressURL            string
	MetaphorDevelopmentIngressURL string
	MetaphorStagingIngressURL     string
	MetaphorProductionIngressURL  string
	GitopsRepoURL                 string
	KubefirstVersion              string
	KubefirstTeam                 string
	UseTelemetry                  string
	GitProvider                   string
	CloudProvider                 string
	ClusterId                     string
	KubeconfigPath                string
}

type MetaphorTokenValues struct {
	ClusterName                   string
	CloudRegion                   string
	ContainerRegistryURL          string
	DomainName                    string
	MetaphorDevelopmentIngressURL string
	MetaphorStagingIngressURL     string
	MetaphorProductionIngressURL  string
}
