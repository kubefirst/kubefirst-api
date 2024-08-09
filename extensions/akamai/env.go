/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/vault"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

func readVaultTokenFromSecret(clientset *kubernetes.Clientset) string {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Printf("Error reading existing Secret data: %s", err)
		return ""
	}

	return existingKubernetesSecret["root-token"]
}

func GetAkamaiTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["LINODE_TOKEN"] = cl.AkamaiAuth.Token
	// needed for s3 api connectivity to object storage
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey
	envs["AWS_SESSION_TOKEN"] = ""        // allows for debugging
	envs["TF_VAR_aws_session_token"] = "" // allows for debugging
	// envs["TF_LOG"] = "debug"

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["GITHUB_TOKEN"] = cl.GitAuth.Token
	envs["GITHUB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey
	envs["AWS_SESSION_TOKEN"] = ""        // allows for debugging
	envs["TF_VAR_aws_session_token"] = "" // allows for debugging

	return envs
}

func GetGitlabTerraformEnvs(envs map[string]string, gid int, cl *pkgtypes.Cluster) map[string]string {
	envs["GITLAB_TOKEN"] = cl.GitAuth.Token
	envs["GITLAB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["AWS_ACCESS_KEY_ID"] = cl.StateStoreCredentials.AccessKeyID
	envs["AWS_SECRET_ACCESS_KEY"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_aws_access_key_id"] = cl.StateStoreCredentials.AccessKeyID
	envs["TF_VAR_aws_secret_access_key"] = cl.StateStoreCredentials.SecretAccessKey
	envs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
	envs["TF_VAR_gitlab_owner"] = cl.GitAuth.Owner
	envs["AWS_SESSION_TOKEN"] = ""        // allows for debugging
	envs["TF_VAR_aws_session_token"] = "" // allows for debugging

	return envs
}

func GetUsersTerraformEnvs(clientset *kubernetes.Clientset, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	envs["AWS_SESSION_TOKEN"] = ""        // allows for debugging
	envs["TF_VAR_aws_session_token"] = "" // allows for debugging

	return envs
}

func GetVaultTerraformEnvs(clientset *kubernetes.Clientset, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	envs["TF_VAR_email_address"] = cl.AlertsEmail
	envs["TF_VAR_vault_addr"] = providerConfigs.VaultPortForwardURL
	envs["TF_VAR_vault_token"] = readVaultTokenFromSecret(clientset)
	envs[fmt.Sprintf("TF_VAR_%s_token", cl.GitProvider)] = cl.GitAuth.Token
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs["VAULT_TOKEN"] = readVaultTokenFromSecret(clientset)
	envs["TF_VAR_akamai_token"] = cl.AkamaiAuth.Token
	envs["LINODE_TOKEN"] = cl.AkamaiAuth.Token
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_private_key"] = cl.GitAuth.PrivateKey
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["TF_VAR_cloudflare_origin_ca_api_key"] = cl.CloudflareAuth.OriginCaIssuerKey
	envs["TF_VAR_cloudflare_api_key"] = cl.CloudflareAuth.APIToken
	envs["AWS_SESSION_TOKEN"] = ""        // allows for debugging
	envs["TF_VAR_aws_session_token"] = "" // allows for debugging

	if cl.GitProvider == "gitlab" {
		envs["TF_VAR_owner_group_id"] = fmt.Sprintf("%d", cl.GitlabOwnerGroupID)
	}

	return envs
}
