/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package k3d

import (
	"os"
	"io/ioutil"
	"strings"
	"path/filepath"
	pkg "github.com/kubefirst/kubefirst-api/internal"
)

func GetGithubTerraformEnvs(config *K3dConfig, envs map[string]string, githubToken string) map[string]string {
	envs["GITHUB_TOKEN"] = config.GithubToken
	envs["AWS_ACCESS_KEY_ID"] = pkg.MinioDefaultUsername
	envs["AWS_SECRET_ACCESS_KEY"] = pkg.MinioDefaultPassword
	envs["TF_VAR_aws_access_key_id"] = pkg.MinioDefaultUsername
	envs["TF_VAR_aws_secret_access_key"] = pkg.MinioDefaultPassword

	return envs
}

func GetUsersTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {
	envs["TF_VAR_email_address"] = "your@email.com"
	envs["TF_VAR_github_token"] = config.GithubToken
	envs["TF_VAR_vault_addr"] = VaultPortForwardURL
	envs["TF_VAR_vault_token"] = "k1_local_vault_token"
	envs["VAULT_ADDR"] = VaultPortForwardURL
	envs["VAULT_TOKEN"] = "k1_local_vault_token"
	envs["GITHUB_TOKEN"] = config.GithubToken

	return envs
}

func GetVaultTerraformEnvs(config *K3dConfig, envs map[string]string) map[string]string {
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
	AwsAccessKeyId        string
	AwsSecretAccessKey    string
}


func TerraformPrep(config *K3dConfig) error{

	path := config.GitopsDir + "/terraform"
	err := filepath.Walk(path,detokenizeterraform(path,config))
	if err != nil {
		return err
	}
	return nil
}

func detokenizeterraform(path string,config *K3dConfig) filepath.WalkFunc {
	return filepath.WalkFunc(func(path string,fi os.FileInfo,err error) error{

		if fi.IsDir()  {
			return nil
		}

		matched,_ := filepath.Match("*",fi.Name())

		if matched {

			read,err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			newContents := string(read)
			newContents = strings.Replace(newContents,"<ADMIN-TEAM>",config.AdminTeamName,-1)
			newContents = strings.Replace(newContents,"<DEVELOPER-TEAM>",config.DeveloperTeamName,-1)
			newContents = strings.Replace(newContents,"<METPAHOR-REPO-NAME>",config.MetaphorRepoName,-1)
			newContents = strings.Replace(newContents,"<GIT-REPO-NAME>",config.GitopsRepoName,-1)

			err = ioutil.WriteFile(path,[]byte(newContents),0)
			if err != nil {
				return err
			}
			
		}
		return nil
	})

}