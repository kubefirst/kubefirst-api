/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package akamai

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/civo/civogo"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	pkg "github.com/kubefirst/kubefirst-api/internal"
	"github.com/kubefirst/kubefirst-api/internal/argocd"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/errors"
	gitlab "github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteAkamaiCluster
func DeleteAkamaiCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {
	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteStarted, "")

	// Instantiate civo config
	config := providerConfigs.GetConfig(cl.ClusterName, cl.DomainName, cl.GitProvider, cl.GitAuth.Owner, cl.GitProtocol, cl.CloudflareAuth.APIToken, cl.CloudflareAuth.OriginCaIssuerKey,cl.GitopsRepoName,cl.MetaphorRepoName)

	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.Status = constants.ClusterStatusDeleting
	err := secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return err
	}

	tfEnvs := map[string]string{}
	var tfEntrypoint string

	if cl.GitTerraformApplyCheck {
		log.Info().Msgf("destroying %s resources with terraform", cl.GitProvider)
		switch cl.GitProvider {
		case "github":
			tfEntrypoint = config.GitopsDir + "/terraform/github"
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
			tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, cl)

		case "gitlab":
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
			if err != nil {
				return err
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			var projectsForDeletion = []string{"gitops", "metaphor"}
			for _, project := range projectsForDeletion {
				projectExists, err := gitlabClient.CheckProjectExists(project)
				if err != nil {
					log.Error().Msgf("could not check for existence of project %s: %s", project, err)
				}
				if projectExists {
					log.Info().Msgf("checking project %s for container registries...", project)
					crr, err := gitlabClient.GetProjectContainerRegistryRepositories(project)
					if err != nil {
						log.Error().Msgf("could not retrieve container registry repositories: %s", err)
					}
					if len(crr) > 0 {
						for _, cr := range crr {
							err := gitlabClient.DeleteContainerRegistryRepository(project, cr.ID)
							if err != nil {
								log.Error().Msgf("error deleting container registry repository: %s", err)
							}
						}
					} else {
						log.Info().Msgf("project %s does not have any container registries, skipping", project)
					}
				} else {
					log.Info().Msgf("project %s does not exist, skipping", project)
				}
			}
			tfEntrypoint = config.GitopsDir + "/terraform/gitlab"
			tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)
			tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
		}

		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Info().Msgf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return err
		}

		log.Info().Msgf("%s resources terraform destroyed", cl.GitProvider)

		cl.GitTerraformApplyCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return err
		}
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			kcfg := k8s.CreateKubeConfig(false, config.Kubeconfig)

			log.Info().Msg("destroying civo resources with terraform")

			client, err := civogo.NewClient(cl.AkamaiAuth.Token, cl.CloudRegion)
			if err != nil {
				return fmt.Errorf(err.Error())
			}

			cluster, err := client.FindKubernetesCluster(cl.ClusterName)
			if err != nil {
				return err
			}
			log.Info().Msg("cluster name: " + cluster.ID)

			clusterVolumes, err := client.ListVolumesForCluster(cluster.ID)
			if err != nil {
				return err
			}

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				log.Info().Msg("opening argocd port forward")
				//* ArgoCD port-forward
				argoCDStopChannel := make(chan struct{}, 1)
				defer func() {
					close(argoCDStopChannel)
				}()
				k8s.OpenPortForwardPodWrapper(
					kcfg.Clientset,
					kcfg.RestConfig,
					"argocd-server",
					"argocd",
					80,
					8080,
					argoCDStopChannel,
				)

				log.Info().Msg("getting new auth token for argocd")

				secData, err := k8s.ReadSecretV2(kcfg.Clientset, "argocd", "argocd-initial-admin-secret")
				if err != nil {
					return err
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return err
				}

				log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				customTransport := http.DefaultTransport.(*http.Transport).Clone()
				customTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
				argocdHttpClient := http.Client{Transport: customTransport}
				log.Info().Msg("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(&argocdHttpClient, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					errors.HandleClusterError(cl, err.Error())
					return err
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			for _, vol := range clusterVolumes {
				log.Info().Msg("removing volume with name: " + vol.Name)
				_, err := client.DeleteVolume(vol.ID)
				if err != nil {
					return err
				}
				log.Info().Msg("volume " + vol.ID + " deleted")
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for Civo Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return err
			}
		}

		log.Info().Msg("destroying civo cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = civoext.GetCivoTerraformEnvs(tfEnvs, cl)

		switch cl.GitProvider {
		case "github":
			tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(fmt.Sprint(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %s", err)
			}
			tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return err
		}
		log.Info().Msg("civo resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return err
		}
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return err
		}
		log.Info().Msg("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn().Msg(err.Error())
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

	cl.Status = constants.ClusterStatusDeleted
	err = secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return err
	}

	err = pkg.ResetK1Dir(config.K1Dir, cl.GitopsRepoName, cl.MetaphorRepoName)
	if err != nil {
		return err
	}

	return nil
}
