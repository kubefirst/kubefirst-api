/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/vault"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
)

func readVaultTokenFromSecret(clientset *kubernetes.Clientset) (string, error) {
	existingKubernetesSecret, err := k8s.ReadSecretV2(clientset, vault.VaultNamespace, vault.VaultSecretName)
	if err != nil || existingKubernetesSecret == nil {
		log.Error().Msgf("Error reading existing Secret data: %s", err)
		return "", fmt.Errorf("error reading existing Secret data: %w", err)
	}

	return existingKubernetesSecret["root-token"], nil
}

func GetGoogleTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["TF_VAR_project"] = cl.GoogleAuth.ProjectID
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	envs["GOOGLE_APPLICATION_CREDENTIALS"] = fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir)

	return envs
}

func GetGithubTerraformEnvs(envs map[string]string, cl *pkgtypes.Cluster) map[string]string {
	envs["GITHUB_TOKEN"] = cl.GitAuth.Token
	envs["GITHUB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	envs["GOOGLE_APPLICATION_CREDENTIALS"] = fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir)

	return envs
}

func GetGitlabTerraformEnvs(envs map[string]string, gid int, cl *pkgtypes.Cluster) map[string]string {
	envs["GITLAB_TOKEN"] = cl.GitAuth.Token
	envs["GITLAB_OWNER"] = cl.GitAuth.Owner
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["TF_VAR_owner_group_id"] = strconv.Itoa(gid)
	envs["TF_VAR_gitlab_owner"] = cl.GitAuth.Owner
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	envs["GOOGLE_APPLICATION_CREDENTIALS"] = fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir)

	return envs
}

func GetUsersTerraformEnvs(clientset *kubernetes.Clientset, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	vaultToken, err := readVaultTokenFromSecret(clientset)
	if err != nil {
		return envs
	}
	envs["VAULT_TOKEN"] = vaultToken
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	envs["GOOGLE_APPLICATION_CREDENTIALS"] = fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir)

	return envs
}

func GetVaultTerraformEnvs(clientset *kubernetes.Clientset, cl *pkgtypes.Cluster, envs map[string]string) map[string]string {
	vaultToken, err := readVaultTokenFromSecret(clientset)
	if err != nil {
		return envs
	}
	envs[fmt.Sprintf("%s_TOKEN", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Token
	envs[fmt.Sprintf("%s_OWNER", strings.ToUpper(cl.GitProvider))] = cl.GitAuth.Owner
	envs["TF_VAR_email_address"] = cl.AlertsEmail
	envs["TF_VAR_vault_addr"] = providerConfigs.VaultPortForwardURL
	envs["TF_VAR_vault_token"] = vaultToken
	envs[fmt.Sprintf("TF_VAR_%s_token", cl.GitProvider)] = cl.GitAuth.Token
	envs["VAULT_ADDR"] = providerConfigs.VaultPortForwardURL
	envs["VAULT_TOKEN"] = vaultToken
	envs["TF_VAR_civo_token"] = cl.CivoAuth.Token
	envs["TF_VAR_atlantis_repo_webhook_secret"] = cl.AtlantisWebhookSecret
	envs["TF_VAR_atlantis_repo_webhook_url"] = cl.AtlantisWebhookURL
	envs["TF_VAR_kbot_ssh_private_key"] = cl.GitAuth.PrivateKey
	envs["TF_VAR_kbot_ssh_public_key"] = cl.GitAuth.PublicKey
	envs["TF_VAR_cloudflare_origin_ca_api_key"] = cl.CloudflareAuth.OriginCaIssuerKey
	envs["TF_VAR_cloudflare_api_key"] = cl.CloudflareAuth.Token
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}
	envs["GOOGLE_APPLICATION_CREDENTIALS"] = fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir)

	if cl.GitProvider == "gitlab" {
		envs["TF_VAR_owner_group_id"] = fmt.Sprint(cl.GitlabOwnerGroupID)
	}

	return envs
}
