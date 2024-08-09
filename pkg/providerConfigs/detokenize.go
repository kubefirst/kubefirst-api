/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs //nolint:revive // allowed during refactoring

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/rs/zerolog/log"
)

// DetokenizeGitGitops - Translate tokens by values on a given path
func DetokenizeGitGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) error {
	fn := detokenizeGitops(tokens, gitProtocol, useCloudflareOriginIssuer)
	err := filepath.Walk(path, fn)
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitops(tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if fi.IsDir() && fi.Name() == ".git" {
			return filepath.SkipDir
		}
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		metaphorDevelopmentIngressURL := fmt.Sprintf("https://metaphor-development.%s", tokens.DomainName)
		metaphorStagingIngressURL := fmt.Sprintf("https://metaphor-staging.%s", tokens.DomainName)
		metaphorProductionIngressURL := fmt.Sprintf("https://metaphor-production.%s", tokens.DomainName)

		// ignore .git files
		if !strings.Contains(path, "/.git/") {
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var fullDomainName string

			if tokens.SubdomainName != "" {
				fullDomainName = fmt.Sprintf("%s.%s", tokens.SubdomainName, tokens.DomainName)
			} else {
				fullDomainName = tokens.DomainName
			}

			newContents := string(read)
			newContents = strings.ReplaceAll(newContents, "<ALERTS_EMAIL>", tokens.AlertsEmail)
			newContents = strings.ReplaceAll(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_NAME>", tokens.ClusterName)
			newContents = strings.ReplaceAll(newContents, "<CLOUD_PROVIDER>", tokens.CloudProvider)
			newContents = strings.ReplaceAll(newContents, "<CLOUD_REGION>", tokens.CloudRegion)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_ID>", tokens.ClusterID)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_TYPE>", tokens.ClusterType)
			newContents = strings.ReplaceAll(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL)
			newContents = strings.ReplaceAll(newContents, "<DOMAIN_NAME>", fullDomainName)
			newContents = strings.ReplaceAll(newContents, "<KUBE_CONFIG_PATH>", tokens.KubeconfigPath)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_ARTIFACTS_BUCKET>", tokens.KubefirstArtifactsBucket)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", tokens.KubefirstStateStoreBucket)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_TEAM>", tokens.KubefirstTeam)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_TEAM_INFO>", os.Getenv("KUBEFIRST_TEAM_INFO"))
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_VERSION>", tokens.KubefirstVersion)
			newContents = strings.ReplaceAll(newContents, "<KUBEFIRST_STATE_STORE_BUCKET_HOSTNAME>", tokens.StateStoreBucketHostname)
			newContents = strings.ReplaceAll(newContents, "<WORKLOAD_CLUSTER_TERRAFORM_MODULE_URL>", tokens.WorkloadClusterTerraformModuleURL)
			newContents = strings.ReplaceAll(newContents, "<WORKLOAD_CLUSTER_BOOTSTRAP_TERRAFORM_MODULE_URL>", tokens.WorkloadClusterBootstrapTerraformModuleURL)
			newContents = strings.ReplaceAll(newContents, "<NODE_TYPE>", tokens.NodeType)
			newContents = strings.ReplaceAll(newContents, "<NODE_COUNT>", strconv.Itoa(tokens.NodeCount))

			// AWS
			newContents = strings.ReplaceAll(newContents, "<AWS_ACCOUNT_ID>", tokens.AwsAccountID)
			newContents = strings.ReplaceAll(newContents, "<AWS_IAM_ARN_ACCOUNT_ROOT>", tokens.AwsIamArnAccountRoot)
			newContents = strings.ReplaceAll(newContents, "<AWS_NODE_CAPACITY_TYPE>", tokens.AwsNodeCapacityType)

			// google
			newContents = strings.ReplaceAll(newContents, "<GOOGLE_PROJECT>", tokens.GoogleProject)
			newContents = strings.ReplaceAll(newContents, "<TERRAFORM_FORCE_DESTROY>", tokens.ForceDestroy)
			newContents = strings.ReplaceAll(newContents, "<GOOGLE_UNIQUENESS>", tokens.GoogleUniqueness)

			if tokens.CloudProvider == "k3s" {
				// k3s
				newContents = strings.ReplaceAll(newContents, "<K3S_ENDPOINT>", tokens.K3sServersPrivateIps[0])
				// TODO: this is a hack to get around
				// need to be refactored into a single function with args
				var terraformServersPrivateIpsList string
				jsonBytes, err := json.Marshal(tokens.K3sServersPrivateIps)
				if err != nil {
					log.Fatal().Msgf("detokenise issue on %s", err)
				}
				terraformServersPrivateIpsList = string(jsonBytes)
				newContents = strings.ReplaceAll(newContents, "<K3S_SERVERS_PRIVATE_IPS>", terraformServersPrivateIpsList)

				var terraformServersPublicIpsList string
				jsonBytes2, err := json.Marshal(tokens.K3sServersPublicIps)
				if err != nil {
					log.Fatal().Msgf("detokenise issue on %s", err)
				}
				terraformServersPublicIpsList = string(jsonBytes2)
				newContents = strings.ReplaceAll(newContents, "<K3S_SERVERS_PUBLIC_IPS>", terraformServersPublicIpsList)

				var terraformServersArgsList string
				jsonBytes3, err := json.Marshal(tokens.K3sServersArgs)
				if err != nil {
					log.Fatal().Msgf("detokenise issue on %s", err)
				}
				terraformServersArgsList = string(jsonBytes3)
				newContents = strings.ReplaceAll(newContents, "<K3S_SERVERS_ARGS>", terraformServersArgsList)

				newContents = strings.ReplaceAll(newContents, "<SSH_USER>", tokens.SSHUser)
			}
			newContents = strings.ReplaceAll(newContents, "<ARGOCD_INGRESS_URL>", tokens.ArgoCDIngressURL)
			newContents = strings.ReplaceAll(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", tokens.ArgoCDIngressNoHTTPSURL)
			newContents = strings.ReplaceAll(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL)
			newContents = strings.ReplaceAll(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", tokens.ArgoWorkflowsIngressNoHTTPSURL)
			newContents = strings.ReplaceAll(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL)
			newContents = strings.ReplaceAll(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", tokens.AtlantisIngressNoHTTPSURL)
			newContents = strings.ReplaceAll(newContents, "<CHARTMUSEUM_INGRESS_URL>", tokens.ChartMuseumIngressURL)
			newContents = strings.ReplaceAll(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL)
			newContents = strings.ReplaceAll(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", tokens.VaultIngressNoHTTPSURL)
			newContents = strings.ReplaceAll(newContents, "<VAULT_DATA_BUCKET>", tokens.VaultDataBucketName)
			newContents = strings.ReplaceAll(newContents, "<VOUCH_INGRESS_URL>", tokens.VouchIngressURL)

			newContents = strings.ReplaceAll(newContents, "<GIT_DESCRIPTION>", tokens.GitDescription)
			newContents = strings.ReplaceAll(newContents, "<GIT_NAMESPACE>", tokens.GitNamespace)
			newContents = strings.ReplaceAll(newContents, "<GIT_PROVIDER>", tokens.GitProvider)
			newContents = strings.ReplaceAll(newContents, "<GIT-PROTOCOL>", gitProtocol)
			newContents = strings.ReplaceAll(newContents, "<GIT_RUNNER>", tokens.GitRunner)
			newContents = strings.ReplaceAll(newContents, "<GIT_RUNNER_DESCRIPTION>", tokens.GitRunnerDescription)
			newContents = strings.ReplaceAll(newContents, "<GIT_RUNNER_NS>", tokens.GitRunnerNS)
			newContents = strings.ReplaceAll(newContents, "<GIT_URL>", tokens.GitURL) // remove

			// GitHub
			newContents = strings.ReplaceAll(newContents, "<GITHUB_HOST>", tokens.GitHubHost)
			newContents = strings.ReplaceAll(newContents, "<GITHUB_OWNER>", strings.ToLower(tokens.GitHubOwner))
			newContents = strings.ReplaceAll(newContents, "<GITHUB_USER>", tokens.GitHubUser)

			// GitLab
			newContents = strings.ReplaceAll(newContents, "<GITLAB_HOST>", tokens.GitlabHost)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_OWNER_GROUP_ID>", strconv.Itoa(tokens.GitlabOwnerGroupID))
			newContents = strings.ReplaceAll(newContents, "<GITLAB_USER>", tokens.GitlabUser)

			newContents = strings.ReplaceAll(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", tokens.GitopsRepoAtlantisWebhookURL)
			newContents = strings.ReplaceAll(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", tokens.GitopsRepoNoHTTPSURL)

			newContents = strings.ReplaceAll(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", metaphorDevelopmentIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressURL)

			// external-dns optionality to provide cloudflare support regardless of cloud provider
			newContents = strings.ReplaceAll(newContents, "<EXTERNAL_DNS_PROVIDER_NAME>", tokens.ExternalDNSProviderName)
			newContents = strings.ReplaceAll(newContents, "<EXTERNAL_DNS_PROVIDER_TOKEN_ENV_NAME>", tokens.ExternalDNSProviderTokenEnvName)
			newContents = strings.ReplaceAll(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_NAME>", tokens.ExternalDNSProviderSecretName)
			newContents = strings.ReplaceAll(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_KEY>", tokens.ExternalDNSProviderSecretKey)
			newContents = strings.ReplaceAll(newContents, "<EXTERNAL_DNS_DOMAIN_NAME>", tokens.DomainName)

			// Catalog
			newContents = strings.ReplaceAll(newContents, "<REGISTRY_PATH>", tokens.RegistryPath)
			newContents = strings.ReplaceAll(newContents, "<SECRET_STORE_REF>", tokens.SecretStoreRef)
			newContents = strings.ReplaceAll(newContents, "<PROJECT>", tokens.Project)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_DESTINATION>", tokens.ClusterDestination)
			newContents = strings.ReplaceAll(newContents, "<ENVIRONMENT>", tokens.Environment)

			// origin issuer defines which annotations should be on ingresses
			if useCloudflareOriginIssuer {
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_1>", "cert-manager.io/issuer: cloudflare-origin-issuer")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_2>", "cert-manager.io/issuer-kind: OriginIssuer")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_3>", "cert-manager.io/issuer-group: cert-manager.k8s.cloudflare.com")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_4>", "external-dns.alpha.kubernetes.io/cloudflare-proxied: \"true\"")
			} else {
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_1>", "cert-manager.io/cluster-issuer: \"letsencrypt-prod\"")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_2>", "")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_3>", "")
				newContents = strings.ReplaceAll(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_4>", "")
			}

			newContents = strings.ReplaceAll(newContents, "<USE_TELEMETRY>", tokens.UseTelemetry)

			// Switch the repo url based on https flag
			newContents = strings.ReplaceAll(newContents, "<GITOPS_REPO_URL>", tokens.GitopsRepoURL)

			// The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
			if gitProtocol == "https" {
				newContents = strings.ReplaceAll(newContents, "<GIT_FQDN>", fmt.Sprintf("https://%v.com/", tokens.GitProvider))
			} else {
				newContents = strings.ReplaceAll(newContents, "<GIT_FQDN>", fmt.Sprintf("git@%v.com:", tokens.GitProvider))
			}

			err = os.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// DetokenizeAdditionalPath - Translate tokens by values on a given path
func DetokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) error {
	fn := detokenizeAdditionalPath(tokens)
	err := filepath.Walk(path, fn)
	if err != nil {
		return err
	}

	return nil
}

// detokenizeAdditionalPath temporary addition to handle detokenizing additional files
func detokenizeAdditionalPath(tokens *GitopsDirectoryValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		// ignore .git files
		if !strings.Contains(path, "/.git/") {
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			newContents := string(read)
			newContents = strings.ReplaceAll(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner)

			err = os.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// DetokenizeGithubMetaphor - Translate tokens by values on a given path
func DetokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {
	fn := detokenizeGitopsMetaphor(tokens)
	err := filepath.Walk(path, fn)
	if err != nil {
		return err
	}
	return nil
}

// DetokenizeDirectoryGithubMetaphor - Translate tokens by values on a directory level.
func detokenizeGitopsMetaphor(tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		// ignore .git files
		if !strings.Contains(path, "/.git/") {
			read, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// todo reduce to terraform tokens by moving to helm chart?
			newContents := string(read)
			newContents = strings.ReplaceAll(newContents, "<CLOUD_REGION>", tokens.CloudRegion)
			newContents = strings.ReplaceAll(newContents, "<CLUSTER_NAME>", tokens.ClusterName)
			newContents = strings.ReplaceAll(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL) // todo need to fix metaphor repo
			newContents = strings.ReplaceAll(newContents, "<DOMAIN_NAME>", tokens.DomainName)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL)
			newContents = strings.ReplaceAll(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL)

			err = os.WriteFile(path, []byte(newContents), 0)
			if err != nil {
				return err
			}
		}

		return nil
	})
}
