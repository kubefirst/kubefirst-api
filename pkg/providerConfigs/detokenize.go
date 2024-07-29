/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package providerConfigs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	log "github.com/rs/zerolog/log"
)

// DetokenizeGitGitops - Translate tokens by values on a given path
func DetokenizeGitGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) error {
	err := filepath.Walk(path, detokenizeGitops(path, tokens, gitProtocol, useCloudflareOriginIssuer))
	if err != nil {
		return err
	}

	return nil
}

func detokenizeGitops(path string, tokens *GitopsDirectoryValues, gitProtocol string, useCloudflareOriginIssuer bool) filepath.WalkFunc {
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

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {

				read, err := ioutil.ReadFile(path)
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
				newContents = strings.Replace(newContents, "<ALERTS_EMAIL>", tokens.AlertsEmail, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_ALLOW_LIST>", tokens.AtlantisAllowList, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CLOUD_PROVIDER>", tokens.CloudProvider, -1)
				newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_ID>", tokens.ClusterId, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_TYPE>", tokens.ClusterType, -1)
				newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1)
				newContents = strings.Replace(newContents, "<DOMAIN_NAME>", fullDomainName, -1)
				newContents = strings.Replace(newContents, "<KUBE_CONFIG_PATH>", tokens.KubeconfigPath, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_ARTIFACTS_BUCKET>", tokens.KubefirstArtifactsBucket, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET>", tokens.KubefirstStateStoreBucket, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_TEAM>", tokens.KubefirstTeam, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_TEAM_INFO>", os.Getenv("KUBEFIRST_TEAM_INFO"), -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_VERSION>", tokens.KubefirstVersion, -1)
				newContents = strings.Replace(newContents, "<KUBEFIRST_STATE_STORE_BUCKET_HOSTNAME>", tokens.StateStoreBucketHostname, -1)
				newContents = strings.Replace(newContents, "<WORKLOAD_CLUSTER_TERRAFORM_MODULE_URL>", tokens.WorkloadClusterTerraformModuleURL, -1)
				newContents = strings.Replace(newContents, "<WORKLOAD_CLUSTER_BOOTSTRAP_TERRAFORM_MODULE_URL>", tokens.WorkloadClusterBootstrapTerraformModuleURL, -1)
				newContents = strings.Replace(newContents, "<NODE_TYPE>", tokens.NodeType, -1)
				newContents = strings.Replace(newContents, "<NODE_COUNT>", fmt.Sprint(tokens.NodeCount), -1)

				// AWS
				newContents = strings.Replace(newContents, "<AWS_ACCOUNT_ID>", tokens.AwsAccountID, -1)
				newContents = strings.Replace(newContents, "<AWS_IAM_ARN_ACCOUNT_ROOT>", tokens.AwsIamArnAccountRoot, -1)
				newContents = strings.Replace(newContents, "<AWS_NODE_CAPACITY_TYPE>", tokens.AwsNodeCapacityType, -1)

				// google
				newContents = strings.Replace(newContents, "<GOOGLE_PROJECT>", tokens.GoogleProject, -1)
				newContents = strings.Replace(newContents, "<TERRAFORM_FORCE_DESTROY>", tokens.ForceDestroy, -1)
				newContents = strings.Replace(newContents, "<GOOGLE_UNIQUENESS>", tokens.GoogleUniqueness, -1)

				if tokens.CloudProvider == "k3s" {
					// k3s
					newContents = strings.Replace(newContents, "<K3S_ENDPOINT>", tokens.K3sServersPrivateIps[0], -1)
					// TODO: this is a hack to get around
					// need to be refactored into a single function with args
					var terraformServersPrivateIpsList string
					jsonBytes, err := json.Marshal(tokens.K3sServersPrivateIps)
					if err != nil {
						log.Fatal().Msgf("detokenise issue on %s", err)
					}
					terraformServersPrivateIpsList = string(jsonBytes)
					newContents = strings.Replace(newContents, "<K3S_SERVERS_PRIVATE_IPS>", terraformServersPrivateIpsList, -1)

					var terraformServersPublicIpsList string
					jsonBytes2, err := json.Marshal(tokens.K3sServersPublicIps)
					if err != nil {
						log.Fatal().Msgf("detokenise issue on %s", err)
					}
					terraformServersPublicIpsList = string(jsonBytes2)
					newContents = strings.Replace(newContents, "<K3S_SERVERS_PUBLIC_IPS>", terraformServersPublicIpsList, -1)

					var terraformServersArgsList string
					jsonBytes3, err := json.Marshal(tokens.K3sServersArgs)
					if err != nil {
						log.Fatal().Msgf("detokenise issue on %s", err)
					}
					terraformServersArgsList = string(jsonBytes3)
					newContents = strings.Replace(newContents, "<K3S_SERVERS_ARGS>", terraformServersArgsList, -1)

					newContents = strings.Replace(newContents, "<SSH_USER>", tokens.SshUser, -1)
				}
				newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_URL>", tokens.ArgoCDIngressURL, -1)
				newContents = strings.Replace(newContents, "<ARGOCD_INGRESS_NO_HTTP_URL>", tokens.ArgoCDIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_URL>", tokens.ArgoWorkflowsIngressURL, -1)
				newContents = strings.Replace(newContents, "<ARGO_WORKFLOWS_INGRESS_NO_HTTPS_URL>", tokens.ArgoWorkflowsIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_URL>", tokens.AtlantisIngressURL, -1)
				newContents = strings.Replace(newContents, "<ATLANTIS_INGRESS_NO_HTTPS_URL>", tokens.AtlantisIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<CHARTMUSEUM_INGRESS_URL>", tokens.ChartMuseumIngressURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_INGRESS_URL>", tokens.VaultIngressURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_INGRESS_NO_HTTPS_URL>", tokens.VaultIngressNoHTTPSURL, -1)
				newContents = strings.Replace(newContents, "<VAULT_DATA_BUCKET>", tokens.VaultDataBucketName, -1)
				newContents = strings.Replace(newContents, "<VOUCH_INGRESS_URL>", tokens.VouchIngressURL, -1)

				newContents = strings.Replace(newContents, "<GIT_DESCRIPTION>", tokens.GitDescription, -1)
				newContents = strings.Replace(newContents, "<GIT_NAMESPACE>", tokens.GitNamespace, -1)
				newContents = strings.Replace(newContents, "<GIT_PROVIDER>", tokens.GitProvider, -1)
				newContents = strings.Replace(newContents, "<GIT-PROTOCOL>", gitProtocol, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER>", tokens.GitRunner, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER_DESCRIPTION>", tokens.GitRunnerDescription, -1)
				newContents = strings.Replace(newContents, "<GIT_RUNNER_NS>", tokens.GitRunnerNS, -1)
				newContents = strings.Replace(newContents, "<GIT_URL>", tokens.GitURL, -1) // remove
				newContents = strings.Replace(newContents, "<ADMIN-TEAM>",tokens.AdminTeamName,-1)
				newContents = strings.Replace(newContents, "<DEVELOPER-TEAM>",tokens.DeveloperTeamName,-1)
				newContents = strings.Replace(newContents, "<METPAHOR-REPO-NAME>",tokens.MetaphorRepoName,-1)
				newContents = strings.Replace(newContents, "<GIT-REPO-NAME>",tokens.GitopsRepoName,-1)

				// GitHub
				newContents = strings.Replace(newContents, "<GITHUB_HOST>", tokens.GitHubHost, -1)
				newContents = strings.Replace(newContents, "<GITHUB_OWNER>", strings.ToLower(tokens.GitHubOwner), -1)
				newContents = strings.Replace(newContents, "<GITHUB_USER>", tokens.GitHubUser, -1)

				// GitLab
				newContents = strings.Replace(newContents, "<GITLAB_HOST>", tokens.GitlabHost, -1)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner, -1)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER_GROUP_ID>", strconv.Itoa(tokens.GitlabOwnerGroupID), -1)
				newContents = strings.Replace(newContents, "<GITLAB_USER>", tokens.GitlabUser, -1)

				newContents = strings.Replace(newContents, "<GITOPS_REPO_ATLANTIS_WEBHOOK_URL>", tokens.GitopsRepoAtlantisWebhookURL, -1)
				newContents = strings.Replace(newContents, "<GITOPS_REPO_NO_HTTPS_URL>", tokens.GitopsRepoNoHTTPSURL, -1)

				newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", metaphorDevelopmentIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", metaphorProductionIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", metaphorStagingIngressURL, -1)

				// external-dns optionality to provide cloudflare support regardless of cloud provider
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_NAME>", tokens.ExternalDNSProviderName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_TOKEN_ENV_NAME>", tokens.ExternalDNSProviderTokenEnvName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_NAME>", tokens.ExternalDNSProviderSecretName, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_PROVIDER_SECRET_KEY>", tokens.ExternalDNSProviderSecretKey, -1)
				newContents = strings.Replace(newContents, "<EXTERNAL_DNS_DOMAIN_NAME>", tokens.DomainName, -1)

				// Catalog
				newContents = strings.Replace(newContents, "<REGISTRY_PATH>", tokens.RegistryPath, -1)
				newContents = strings.Replace(newContents, "<SECRET_STORE_REF>", tokens.SecretStoreRef, -1)
				newContents = strings.Replace(newContents, "<PROJECT>", tokens.Project, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_DESTINATION>", tokens.ClusterDestination, -1)
				newContents = strings.Replace(newContents, "<ENVIRONMENT>", tokens.Environment, -1)

				//origin issuer defines which annotations should be on ingresses
				if useCloudflareOriginIssuer {
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_1>", "cert-manager.io/issuer: cloudflare-origin-issuer", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_2>", "cert-manager.io/issuer-kind: OriginIssuer", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_3>", "cert-manager.io/issuer-group: cert-manager.k8s.cloudflare.com", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_4>", "external-dns.alpha.kubernetes.io/cloudflare-proxied: \"true\"", -1)
				} else {
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_1>", "cert-manager.io/cluster-issuer: \"letsencrypt-prod\"", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_2>", "", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_3>", "", -1)
					newContents = strings.Replace(newContents, "<CERT_MANAGER_ISSUER_ANNOTATION_4>", "", -1)
				}

				newContents = strings.Replace(newContents, "<USE_TELEMETRY>", tokens.UseTelemetry, -1)

				// Switch the repo url based on https flag
				newContents = strings.Replace(newContents, "<GITOPS_REPO_URL>", tokens.GitopsRepoURL, -1)

				// The fqdn is used by metaphor/argo to choose the appropriate url for cicd operations.
				if gitProtocol == "https" {
					newContents = strings.Replace(newContents, "<GIT_FQDN>", fmt.Sprintf("https://%v.com/", tokens.GitProvider), -1)
				} else {
					newContents = strings.Replace(newContents, "<GIT_FQDN>", fmt.Sprintf("git@%v.com:", tokens.GitProvider), -1)
				}

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DetokenizeAdditionalPath - Translate tokens by values on a given path
func DetokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) error {
	err := filepath.Walk(path, detokenizeAdditionalPath(path, tokens))
	if err != nil {
		return err
	}

	return nil
}

// detokenizeAdditionalPath temporary addition to handle detokenizing additional files
func detokenizeAdditionalPath(path string, tokens *GitopsDirectoryValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {
				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				newContents := string(read)
				newContents = strings.Replace(newContents, "<GITLAB_OWNER>", tokens.GitlabOwner, -1)

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// DetokenizeGithubMetaphor - Translate tokens by values on a given path
func DetokenizeGitMetaphor(path string, tokens *MetaphorTokenValues) error {
	err := filepath.Walk(path, detokenizeGitopsMetaphor(path, tokens))
	if err != nil {
		return err
	}
	return nil
}

// DetokenizeDirectoryGithubMetaphor - Translate tokens by values on a directory level.
func detokenizeGitopsMetaphor(path string, tokens *MetaphorTokenValues) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !!fi.IsDir() {
			return nil
		}

		// var matched bool
		matched, _ := filepath.Match("*", fi.Name())

		if matched {
			// ignore .git files
			if !strings.Contains(path, "/.git/") {

				read, err := ioutil.ReadFile(path)
				if err != nil {
					return err
				}

				// todo reduce to terraform tokens by moving to helm chart?
				newContents := string(read)
				newContents = strings.Replace(newContents, "<CLOUD_REGION>", tokens.CloudRegion, -1)
				newContents = strings.Replace(newContents, "<CLUSTER_NAME>", tokens.ClusterName, -1)
				newContents = strings.Replace(newContents, "<CONTAINER_REGISTRY_URL>", tokens.ContainerRegistryURL, -1) // todo need to fix metaphor repo names
				newContents = strings.Replace(newContents, "<DOMAIN_NAME>", tokens.DomainName, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_DEVELOPMENT_INGRESS_URL>", tokens.MetaphorDevelopmentIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_PRODUCTION_INGRESS_URL>", tokens.MetaphorProductionIngressURL, -1)
				newContents = strings.Replace(newContents, "<METAPHOR_STAGING_INGRESS_URL>", tokens.MetaphorStagingIngressURL, -1)

				err = ioutil.WriteFile(path, []byte(newContents), 0)
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}
