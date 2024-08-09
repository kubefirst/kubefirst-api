/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"fmt"
	"os"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/controller"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/pkg/google"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
)

func CreateGoogleCluster(definition *pkgtypes.ClusterDefinition) error {
	ctrl := controller.ClusterController{}
	err := ctrl.InitController(definition)
	if err != nil {
		return err
	}

	// Update cluster status in database

	ctrl.Cluster.InProgress = true
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return err
	}

	// TODO Validate Google region
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error getting home path: %s", err)
	}

	err = google.WriteGoogleApplicationCredentialsFile(definition.GoogleAuth.KeyFile, homeDir)
	if err != nil {
		log.Fatal().Msgf("error writing google application credentials file: %s", err)
	}

	os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", fmt.Sprintf("%s/.k1/application-default-credentials.json", homeDir))

	err = ctrl.DownloadTools(ctrl.ProviderConfig.ToolsDir)
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.DomainLivenessTest()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.StateStoreCredentials()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// Checks for existing repos
	err = ctrl.GitInit()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.InitializeBot()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// Where detokeinization happens
	err = ctrl.RepositoryPrep()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.RunGitTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.RepositoryPush()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.CreateCluster()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.DetokenizeKMSKeyID()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	kcfg, err := ctrl.GoogleClient.GetContainerClusterAuth(ctrl.ClusterName, []byte(ctrl.GoogleAuth.KeyFile))
	if err != nil {
		return err
	}
	// Save config
	ctrl.Kcfg = kcfg

	err = ctrl.WaitForClusterReady()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.InstallArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.InitializeArgoCD()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// Needs wait after cluster create
	ctrl.Cluster.InProgress = false
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return err
	}

	err = ctrl.ClusterSecretsBootstrap()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	ctrl.Cluster.ClusterSecretsCreatedCheck = true
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		ctrl.Cluster.InProgress = false
		err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
		if err != nil {
			return err
		}

		return err
	}

	err = ctrl.DeployRegistryApplication()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.WaitForVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// * configure vault with terraform
	// * vault port-forward
	vaultStopChannel := make(chan struct{}, 1)
	defer func() {
		close(vaultStopChannel)
	}()
	k8s.OpenPortForwardPodWrapper(
		kcfg.Clientset,
		kcfg.RestConfig,
		"vault-0",
		"vault",
		8200,
		8200,
		vaultStopChannel,
	)

	err = ctrl.InitializeVault()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.RunVaultTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.WriteVaultSecrets()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	err = ctrl.RunUsersTerraform()
	if err != nil {
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// Wait for last sync wave app transition to Running
	log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
	crossplaneDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/instance",
		"crossplane",
		"crossplane-system",
		3600,
	)
	if err != nil {
		log.Error().Msgf("Error finding crossplane Deployment: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}
	log.Info().Msg("waiting on dns, tls certificates from letsencrypt and remaining sync waves.\n this may take up to 60 minutes but regularly completes in under 20 minutes")
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, crossplaneDeployment, 3600)
	if err != nil {
		log.Error().Msgf("Error waiting for all Apps to sync ready state: %s", err)

		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	stopCh := make(chan struct{}, 1)
	defer func() {
		close(stopCh)
	}()

	// * export and import cluster
	err = ctrl.ExportClusterRecord()
	if err != nil {
		log.Error().Msgf("Error exporting cluster record: %s", err)
		return err
	}

	ctrl.Cluster.Status = constants.ClusterStatusProvisioned
	ctrl.Cluster.InProgress = false
	err = secrets.UpdateCluster(ctrl.KubernetesClient, ctrl.Cluster)
	if err != nil {
		return err
	}

	log.Info().Msg("cluster creation complete")

	// Create default service entries
	cl, _ := secrets.GetCluster(ctrl.KubernetesClient, ctrl.ClusterName)
	err = services.AddDefaultServices(&cl)
	if err != nil {
		log.Error().Msgf("error adding default service entries for cluster %s: %s", cl.ClusterName, err)
	}

	log.Info().Msg("waiting for kubefirst-api Deployment to transition to Running")
	kubefirstAPI, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"kubefirst-api",
		"kubefirst",
		1200,
	)
	if err != nil {
		log.Error().Msgf("Error finding kubefirst api Deployment: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, kubefirstAPI, 300)
	if err != nil {
		log.Error().Msgf("Error waiting for kubefirst-api to transition to Running: %s", err)

		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	// Wait for last sync wave app transition to Running
	log.Info().Msg("waiting for final sync wave Deployment to transition to Running")
	argocdDeployment, err := k8s.ReturnDeploymentObject(
		kcfg.Clientset,
		"app.kubernetes.io/name",
		"argocd-server",
		"argocd",
		3600,
	)
	if err != nil {
		log.Error().Msgf("Error finding argocd Deployment: %s", err)
		ctrl.UpdateClusterOnError(err.Error())
		return err
	}
	_, err = k8s.WaitForDeploymentReady(kcfg.Clientset, argocdDeployment, 3600)
	if err != nil {
		log.Error().Msgf("Error waiting for argocd deployment to enter Ready state: %s", err)

		ctrl.UpdateClusterOnError(err.Error())
		return err
	}

	log.Info().Msg("cluster creation complete")

	return nil
}
