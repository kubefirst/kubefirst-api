/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package pkg

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/thanhpk/randstr"
)

func CreateTokensFromDatabaseRecord(cl *pkgtypes.Cluster, registryPath string, secretStoreRef string, project string, clusterDestination string, environment string, clusterName string) *providerConfigs.GitopsDirectoryValues {
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	var fullDomainName string
	if cl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", cl.SubdomainName, cl.DomainName)
	} else {
		fullDomainName = cl.DomainName
	}

	destinationGitopsRepoURL := fmt.Sprintf("https://%s/%s/gitops.git", cl.GitHost, cl.GitAuth.Owner)

	if cl.GitProtocol == "ssh" {
		destinationGitopsRepoURL = fmt.Sprintf("git@%s:%s/gitops.git", cl.GitHost, cl.GitAuth.Owner)
	}

	var externalDNSProviderTokenEnvName, externalDNSProviderSecretKey string
	if cl.DNSProvider == "cloudflare" {
		externalDNSProviderTokenEnvName = "CF_API_TOKEN"
		externalDNSProviderSecretKey = "cf-api-token"
	} else {
		switch cl.CloudProvider {
		// provider auth secret gets mapped to these values
		case "aws":
			externalDNSProviderTokenEnvName = "not-used-uses-service-account"
		case "google":
			// Normally this would be GOOGLE_APPLICATION_CREDENTIALS but we are using a service account instead and
			// if you set externalDNSProviderTokenEnvName to GOOGLE_APPLICATION_CREDENTIALS then externaldns will overlook the service account
			// if you want to use the provided keyfile instead of a service account then set the var accordingly
			externalDNSProviderTokenEnvName = fmt.Sprintf("%s_auth", strings.ToUpper(cl.CloudProvider))
		case "civo":
			externalDNSProviderTokenEnvName = fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.CloudProvider))
		case "vultr":
			externalDNSProviderTokenEnvName = fmt.Sprintf("%s_API_KEY", strings.ToUpper(cl.CloudProvider))
		case "digitalocean":
			externalDNSProviderTokenEnvName = "DO_TOKEN"
		}
		externalDNSProviderSecretKey = fmt.Sprintf("%s-auth", cl.CloudProvider)
	}

	containerRegistryHost := "ghcr.io"

	if cl.GitProvider == "gitlab" {
		containerRegistryHost = "registry.gitlab.com"
	}

	// Updating cluster name for workload clusters
	clusterNameToken := cl.ClusterName

	if clusterName != "" {
		clusterNameToken = clusterName
	}

	// Default gitopsTemplateTokens
	gitopsTemplateTokens := &providerConfigs.GitopsDirectoryValues{
		AlertsEmail:                    cl.AlertsEmail,
		AtlantisAllowList:              fmt.Sprintf("%s/%s/*", cl.GitHost, cl.GitAuth.Owner),
		CloudProvider:                  cl.CloudProvider,
		CloudRegion:                    cl.CloudRegion,
		ClusterName:                    clusterNameToken,
		ClusterType:                    cl.ClusterType,
		DomainName:                     cl.DomainName,
		SubdomainName:                  cl.SubdomainName,
		KubefirstTeam:                  cl.KubefirstTeam,
		NodeType:                       cl.NodeType,
		NodeCount:                      cl.NodeCount,
		KubefirstVersion:               env.KubefirstVersion,
		ArgoCDIngressURL:               fmt.Sprintf("https://argocd.%s", fullDomainName),
		ArgoCDIngressNoHTTPSURL:        fmt.Sprintf("argocd.%s", fullDomainName),
		ArgoWorkflowsIngressURL:        fmt.Sprintf("https://argo.%s", fullDomainName),
		ArgoWorkflowsIngressNoHTTPSURL: fmt.Sprintf("argo.%s", fullDomainName),
		AtlantisIngressURL:             fmt.Sprintf("https://atlantis.%s", fullDomainName),
		AtlantisIngressNoHTTPSURL:      fmt.Sprintf("atlantis.%s", fullDomainName),
		ChartMuseumIngressURL:          fmt.Sprintf("https://chartmuseum.%s", fullDomainName),
		VaultIngressURL:                fmt.Sprintf("https://vault.%s", fullDomainName),
		VaultIngressNoHTTPSURL:         fmt.Sprintf("vault.%s", fullDomainName),
		VouchIngressURL:                fmt.Sprintf("https://vouch.%s", fullDomainName),
		RegistryPath:                   registryPath,
		SecretStoreRef:                 secretStoreRef,
		Project:                        project,
		ClusterDestination:             clusterDestination,
		Environment:                    environment,

		GitDescription:       fmt.Sprintf("%s hosted git", cl.GitProvider),
		GitNamespace:         "N/A",
		GitProvider:          cl.GitProvider,
		GitRunner:            fmt.Sprintf("%s Runner", cl.GitProvider),
		GitRunnerDescription: fmt.Sprintf("Self Hosted %s Runner", cl.GitProvider),
		GitRunnerNS:          fmt.Sprintf("%s-runner", cl.GitProvider),
		GitURL:               cl.GitopsTemplateURL,
		GitopsRepoURL:        destinationGitopsRepoURL,

		GitHubHost:  fmt.Sprintf("https://github.com/%s/gitops.git", cl.GitAuth.Owner),
		GitHubOwner: cl.GitAuth.Owner,
		GitHubUser:  cl.GitAuth.User,

		GitlabHost:         cl.GitHost,
		GitlabOwner:        cl.GitAuth.Owner,
		GitlabOwnerGroupID: cl.GitlabOwnerGroupID,
		GitlabUser:         cl.GitAuth.User,

		GitopsRepoAtlantisWebhookURL:               cl.AtlantisWebhookURL,
		GitopsRepoNoHTTPSURL:                       fmt.Sprintf("%s/%s/gitops.git", cl.GitHost, cl.GitAuth.Owner),
		WorkloadClusterTerraformModuleURL:          fmt.Sprintf("git::https://%s/%s/gitops.git//terraform/%s/modules/workload-cluster?ref=main", cl.GitHost, cl.GitAuth.Owner, cl.CloudProvider),
		WorkloadClusterBootstrapTerraformModuleURL: fmt.Sprintf("git::https://%s/%s/gitops.git//terraform/%s/modules/bootstrap?ref=main", cl.GitHost, cl.GitAuth.Owner, cl.CloudProvider),
		ClusterID: cl.ClusterID,

		// external-dns optionality to provide cloudflare support regardless of cloud provider
		ExternalDNSProviderName:         cl.DNSProvider,
		ExternalDNSProviderTokenEnvName: externalDNSProviderTokenEnvName,
		ExternalDNSProviderSecretName:   fmt.Sprintf("%s-auth", cl.CloudProvider),
		ExternalDNSProviderSecretKey:    externalDNSProviderSecretKey,

		ContainerRegistryURL: fmt.Sprintf("%s/%s", containerRegistryHost, cl.GitAuth.Owner), // Not Supported for AWS ECR
	}

	// Handle provider specific tokens
	switch cl.CloudProvider {
	case "vultr":
		gitopsTemplateTokens.StateStoreBucketHostname = cl.StateStoreDetails.Hostname
	case "google":
		gitopsTemplateTokens.GoogleAuth = cl.GoogleAuth.KeyFile
		gitopsTemplateTokens.GoogleProject = cl.GoogleAuth.ProjectID
		gitopsTemplateTokens.GoogleUniqueness = strings.ToLower(randstr.String(5))
		gitopsTemplateTokens.ForceDestroy = strconv.FormatBool(true) // TODO make this optional
		gitopsTemplateTokens.KubefirstArtifactsBucket = cl.StateStoreDetails.Name
		gitopsTemplateTokens.VaultDataBucketName = fmt.Sprintf("%s-vault-data-%s", cl.GoogleAuth.ProjectID, cl.ClusterName)
	case "aws":
		gitopsTemplateTokens.KubefirstArtifactsBucket = cl.StateStoreDetails.Name
		gitopsTemplateTokens.AtlantisWebhookURL = cl.AtlantisWebhookURL
	}

	return gitopsTemplateTokens
}
