/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	vaultapi "github.com/hashicorp/vault/api"
	akamaiext "github.com/kubefirst/kubefirst-api/extensions/akamai"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	k3sext "github.com/kubefirst/kubefirst-api/extensions/k3s"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	vault "github.com/kubefirst/kubefirst-api/internal/vault"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InitializeVault
func (clctrl *ClusterController) GetUserPassword(user string) error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	// empty conf
	vaultConf := &vault.Conf
	// sets up vault client within function
	clctrl.VaultAuth.KbotPassword, err = vaultConf.GetUserPassword(vault.VaultDefaultAddress, cl.VaultAuth.RootToken, user, "initial-password")
	if err != nil {
		return err
	}

	clctrl.Cluster.VaultAuth.KbotPassword = clctrl.VaultAuth.KbotPassword
	err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
	if err != nil {
		return err
	}

	return nil
}

// InitializeVault
func (clctrl *ClusterController) InitializeVault() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.VaultInitializedCheck {
		var kcfg *k8s.KubernetesClient
		var vaultHandlerPath string

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
			vaultHandlerPath = "github.com:kubefirst/manifests.git/vault-handler/replicas-3"
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationStarted, "")

		switch clctrl.CloudProvider {
		case "aws", "google":
			vaultClient := &vault.Conf

			initResponse, err := vaultClient.AutoUnseal()
			if err != nil {
				return err
			}

			vaultRootToken := initResponse.RootToken

			dataToWrite := make(map[string][]byte)
			dataToWrite["root-token"] = []byte(vaultRootToken)
			for i, value := range initResponse.Keys {
				dataToWrite[fmt.Sprintf("root-unseal-key-%v", i+1)] = []byte(value)
			}
			secret := v1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      vault.VaultSecretName,
					Namespace: vault.VaultNamespace,
				},
				Data: dataToWrite,
			}

			err = k8s.CreateSecretV2(kcfg.Clientset, &secret)
			if err != nil {
				return err
			}
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			// Initialize and unseal Vault
			// Build and apply manifests
			yamlData, err := kcfg.KustomizeBuild(vaultHandlerPath)
			if err != nil {
				return err
			}
			output, err := kcfg.SplitYAMLFile(yamlData)
			if err != nil {
				return err
			}
			err = kcfg.ApplyObjects(output)
			if err != nil {
				return err
			}

			// Wait for the Job to finish
			job, err := k8s.ReturnJobObject(kcfg.Clientset, "vault", "vault-handler")
			if err != nil {
				return err
			}
			_, err = k8s.WaitForJobComplete(kcfg.Clientset, job.Name, job.Namespace, 240)
			if err != nil {
				msg := fmt.Sprintf("could not run vault unseal job: %s", err)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationFailed, err.Error())
				log.Error().Msg(msg)
			}
		}
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultInitializationCompleted, "")

		clctrl.Cluster.VaultInitializedCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunVaultTerraform
func (clctrl *ClusterController) RunVaultTerraform() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.VaultTerraformApplyCheck {
		var kcfg *k8s.KubernetesClient

		switch clctrl.CloudProvider {
		case "aws":
			kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
		case "akamai", "civo", "digitalocean", "k3s", "vultr":
			kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
		case "google":
			var err error
			kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
			if err != nil {
				return err
			}
		}

		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyStarted, "")

		tfEnvs := map[string]string{}

		// Common TfEnvs
		var usernamePasswordString, base64DockerAuth, registryAuth string

		if clctrl.GitProvider == "gitlab" {
			registryAuth, err = clctrl.ContainerRegistryAuth()
			if err != nil {
				return err
			}

			usernamePasswordString = fmt.Sprintf("%s:%s", "container-registry-auth", registryAuth)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		} else {
			usernamePasswordString = fmt.Sprintf("%s:%s", clctrl.GitAuth.User, clctrl.GitAuth.Token)
			base64DockerAuth = base64.StdEncoding.EncodeToString([]byte(usernamePasswordString))
		}

		tfEnvs["TF_VAR_b64_docker_auth"] = base64DockerAuth

		if clctrl.GitProvider == "gitlab" {
			tfEnvs["TF_VAR_container_registry_auth"] = registryAuth
			tfEnvs["TF_VAR_owner_group_id"] = strconv.Itoa(clctrl.GitlabOwnerGroupID)
		}

		// Specific TfEnvs
		switch clctrl.CloudProvider {
		case "akamai":
			tfEnvs = akamaiext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = akamaiext.GetAkamaiTerraformEnvs(tfEnvs, &cl)
		case "aws":
			tfEnvs = awsext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = awsext.GetAwsTerraformEnvs(tfEnvs, &cl)
		case "civo":
			tfEnvs = civoext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, &cl)
		case "google":
			tfEnvs = googleext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, &cl)
		case "digitalocean":
			tfEnvs = digitaloceanext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, &cl)
		case "vultr":
			tfEnvs = vultrext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = vultrext.GetVultrTerraformEnvs(tfEnvs, &cl)
		case "k3s":
			tfEnvs = k3sext.GetVaultTerraformEnvs(kcfg.Clientset, &cl, tfEnvs)
			tfEnvs = k3sext.GetK3sTerraformEnvs(tfEnvs, &cl)
		}

		tfEntrypoint := clctrl.ProviderConfig.GitopsDir + "/terraform/vault"
		terraformClient := clctrl.ProviderConfig.TerraformClient

		log.Info().Msg("configuring vault with terraform")
		err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error applying vault terraform: %s", err)
			log.Info().Msg("sleeping 10 seconds before retrying terraform execution once more")
			time.Sleep(10 * time.Second)
			err = terraformext.InitApplyAutoApprove(terraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error applying vault terraform: %s", err)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyFailed, err.Error())
				return err
			}
		}

		log.Info().Msg("vault terraform executed successfully")
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.VaultTerraformApplyCompleted, "")

		clctrl.Cluster.VaultTerraformApplyCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clctrl *ClusterController) WriteVaultSecrets() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	vaultAddr := "http://localhost:8200"

	vaultClient, err := vaultapi.NewClient(&vaultapi.Config{
		Address: vaultAddr,
	})
	if err != nil {
		log.Error().Msgf("error creating vault client: %s", err)
		return err
	}

	var externalDNSToken string
	switch cl.DNSProvider {
	case "akamai":
		externalDNSToken = cl.AkamaiAuth.Token
	case "civo":
		externalDNSToken = cl.CivoAuth.Token
	case "vultr":
		externalDNSToken = cl.VultrAuth.Token
	case "digitalocean":
		externalDNSToken = cl.DigitaloceanAuth.Token
	case "aws":
		externalDNSToken = "implement with cluster management"
	case "google":
		externalDNSToken = "implement with cluster management"
	case "cloudflare":
		externalDNSToken = cl.CloudflareAuth.APIToken
	}
	//
	var kcfg *k8s.KubernetesClient
	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "akamai", "civo", "digitalocean", "k3s", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return err
		}
	}

	clientset := kcfg.Clientset

	var vaultRootToken string
	vaultUnsealSecretData, err := k8s.ReadSecretV2(clientset, "vault", "vault-unseal-secret")
	if err != nil {
		log.Error().Msgf("error reading vault-unseal-secret: %s", err)
	}
	if len(vaultUnsealSecretData) != 0 {
		vaultRootToken = vaultUnsealSecretData["root-token"]
	}
	vaultClient.SetToken(vaultRootToken)

	if clctrl.CloudProvider == "akamai" {
		secretToCreate := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "vault-secrets",
				Namespace: "external-secrets-operator",
			},
			Data: map[string][]byte{
				"vault-token": []byte(vaultRootToken),
			},
		}
		k8s.CreateSecretV2(kcfg.Clientset, secretToCreate)
	}

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "external-dns", map[string]interface{}{
		"token": externalDNSToken,
	})
	if err != nil {
		log.Error().Msgf("error writing secret to vault: %s", err)
		return err
	}

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "cloudflare", map[string]interface{}{
		"origin-ca-api-key": cl.CloudflareAuth.OriginCaIssuerKey,
	})
	if err != nil {
		log.Error().Msgf("error writing secret to vault: %s", err)
		return err
	}

	// _, err = vaultClient.KVv2("secret").Put(context.Background(), "crossplane", map[string]interface{}{
	// 	"username": cl.GitAuth.User,
	// 	"password": cl.GitAuth.Token,
	// })

	if cl.CloudProvider == "google" {
		log.Info().Msg("writing google specific secrets to vault secret store")
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatal().Msgf("error getting home path: %s", err)
		}
		if err := writeGoogleSecrets(homeDir, vaultClient); err != nil {
			log.Error().Msgf("error writing Google secrets to vault: %s", err)
			return err
		}
		log.Info().Msg("successfully wrote google specific secrets to vault")
	}

	log.Info().Msg("successfully wrote platform secrets to vault secret store")
	return nil
}

// WaitForVault
func (clctrl *ClusterController) WaitForVault() error {
	var kcfg *k8s.KubernetesClient

	switch clctrl.CloudProvider {
	case "aws":
		kcfg = awsext.CreateEKSKubeconfig(&clctrl.AwsClient.Config, clctrl.ClusterName)
	case "akamai", "civo", "digitalocean", "k3s", "vultr":
		kcfg = k8s.CreateKubeConfig(false, clctrl.ProviderConfig.Kubeconfig)
	case "google":
		var err error
		kcfg, err = clctrl.GoogleClient.GetContainerClusterAuth(clctrl.ClusterName, []byte(clctrl.GoogleAuth.KeyFile))
		if err != nil {
			return err
		}
	}

	vaultStatefulSet, err := k8s.ReturnStatefulSetObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"vault",
		"vault",
		1200,
	)
	if err != nil {
		log.Error().Msgf("error finding Vault StatefulSet: %s", err)
		return err
	}
	_, err = k8s.WaitForStatefulSetReady(kcfg.Clientset, vaultStatefulSet, 300, true)
	if err != nil {
		log.Error().Msgf("error waiting for Vault StatefulSet ready state: %s", err)
		return err
	}

	return nil
}

func writeGoogleSecrets(homeDir string, vaultClient *vaultapi.Client) error {
	// vault path - gcp/application-default-credentials
	adcJSON, err := os.ReadFile(fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))
	if err != nil {
		log.Error().Msg("error: reading google json credentials file")
		return err
	}

	var data map[string]interface{}
	err = json.Unmarshal([]byte(adcJSON), &data)
	if err != nil {
		return err
	}

	data["private_key"] = strings.ReplaceAll(data["private_key"].(string), "\n", "\\n")

	_, err = vaultClient.KVv2("secret").Put(context.Background(), "gcp/application-default-credentials", data)
	if err != nil {
		return err
	}
	return nil
}
