/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	pkg "github.com/konstructio/kubefirst-api/internal"
)

func GetGithubTerraformEnvs(config *Config, envs map[string]string) map[string]string {
	if envs == nil {
		envs = make(map[string]string)
	}

	envs["GITHUB_TOKEN"] = config.GithubToken
	envs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
	envs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
	envs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	envs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

	return envs
}

func GetUsersTerraformEnvs(config *Config, envs map[string]string) map[string]string {
	if envs == nil {
		envs = make(map[string]string)
	}

	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = config.GithubToken
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["GITHUB_TOKEN"] = config.GithubToken

	return envs
}

func GetVaultTerraformEnvs(config *Config, envs map[string]string) map[string]string {
	if envs == nil {
		envs = make(map[string]string)
	}

	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = config.GithubToken
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	envs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

	return envs
}

type GithubTerraformEnvs struct {
	GithubToken           string
	GithubOwner           string
	AtlantisWebhookSecret string
	KbotSSHPublicKey      string
	AwsAccessKeyID        string
	AwsSecretAccessKey    string
}
