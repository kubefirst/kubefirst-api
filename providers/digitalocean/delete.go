/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package digitalocean

import (
	"context"
	"fmt"
	"strconv"
	"time"

	digitaloceanext "github.com/konstructio/kubefirst-api/extensions/digitalocean"
	terraformext "github.com/konstructio/kubefirst-api/extensions/terraform"
	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/argocd"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/digitalocean"
	"github.com/konstructio/kubefirst-api/internal/errors"
	gitlab "github.com/konstructio/kubefirst-api/internal/gitlab"
	"github.com/konstructio/kubefirst-api/internal/httpCommon"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/utils"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteDigitaloceanCluster
func DeleteDigitaloceanCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {
	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteStarted, "")

	// Instantiate provider config
	config, err := providerConfigs.GetConfig(
		cl.ClusterName,
		cl.DomainName,
		cl.GitProvider,
		cl.GitAuth.Owner,
		cl.GitProtocol,
		cl.CloudflareAuth.APIToken,
		cl.CloudflareAuth.OriginCaIssuerKey,
	)
	if err != nil {
		return fmt.Errorf("error getting provider config: %w", err)
	}

	kcfg := utils.GetKubernetesClient(cl.ClusterName)

	cl.Status = constants.ClusterStatusDeleting

	if err := secrets.UpdateCluster(kcfg.Clientset, *cl); err != nil {
		return fmt.Errorf("error updating cluster: %w", err)
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, cl)
			tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Printf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("error executing terraform destroy %s: %w", tfEntrypoint, err)
			}
			log.Info().Msg("github resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster: %w", err)
			}
		}
	case "gitlab":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying gitlab resources with terraform")
			gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
			if err != nil {
				return fmt.Errorf("error creating new GitLab client: %w", err)
			}

			// Before removing Terraform resources, remove any container registry repositories
			// since failing to remove them beforehand will result in an apply failure
			projectsForDeletion := []string{"gitops", "metaphor"}
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

			tfEntrypoint := config.GitopsDir + "/terraform/gitlab"
			tfEnvs := map[string]string{}
			tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, cl)
			tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Info().Msgf("error executing terraform destroy %s", tfEntrypoint)
				errors.HandleClusterError(cl, err.Error())
				return fmt.Errorf("error executing terraform destroy %s: %w", tfEntrypoint, err)
			}

			log.Info().Msg("gitlab resources terraform destroyed")

			cl.GitTerraformApplyCheck = false
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster: %w", err)
			}
		}
	}

	// Should be a "cluster was created" check
	if cl.CloudTerraformApplyCheck {
		kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
		if err != nil {
			return fmt.Errorf("error creating kubeconfig: %w", err)
		}

		// Remove applications with external dependencies
		removeArgoCDApps := []string{
			"ingress-nginx-components",
			"ingress-nginx",
			"argo-components",
			"argo",
			"atlantis-components",
			"atlantis",
			"vault-components",
			"vault",
		}
		err = argocd.ApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
		if err != nil {
			log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
		}
		// Pause before cluster destroy to prevent a race condition
		log.Info().Msg("waiting for argocd application deletion to complete...")
		time.Sleep(time.Second * 20)
	}

	// Fetch cluster resources prior to deletion
	digitaloceanConf := digitalocean.Configuration{
		Client:  digitalocean.NewDigitalocean(cl.DigitaloceanAuth.Token),
		Context: context.Background(),
	}
	resources, err := digitaloceanConf.GetKubernetesAssociatedResources(cl.ClusterName)
	if err != nil {
		return fmt.Errorf("error getting kubernetes associated resources: %w", err)
	}

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			kcfg, err := k8s.CreateKubeConfig(false, config.Kubeconfig)
			if err != nil {
				return fmt.Errorf("error creating kubeconfig: %w", err)
			}

			log.Info().Msg("destroying digitalocean resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				log.Info().Msg("opening argocd port forward")
				// * ArgoCD port-forward
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
					return fmt.Errorf("error reading argocd initial admin secret: %w", err)
				}
				argocdPassword := secData["password"]

				argocdAuthToken, err := argocd.GetArgoCDToken("admin", argocdPassword)
				if err != nil {
					return fmt.Errorf("error getting argocd token: %w", err)
				}

				log.Info().Msgf("port-forward to argocd is available at %s", providerConfigs.ArgocdPortForwardURL)

				client := httpCommon.CustomHTTPClient(true)
				log.Info().Msg("deleting the registry application")
				httpCode, _, err := argocd.DeleteApplication(client, config.RegistryAppName, argocdAuthToken, "true")
				if err != nil {
					errors.HandleClusterError(cl, err.Error())
					return fmt.Errorf("error deleting the registry application: %w", err)
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for digitalocean kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster: %w", err)
			}
		}

		log.Info().Msg("destroying digitalocean cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = digitaloceanext.GetDigitaloceanTerraformEnvs(tfEnvs, cl)

		switch cl.GitProvider {
		case "github":
			tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(strconv.Itoa(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %w", err)
			}
			tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Printf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return fmt.Errorf("error executing terraform destroy %s: %w", tfEntrypoint, err)
		}
		log.Info().Msg("digitalocean resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return fmt.Errorf("error updating cluster: %w", err)
		}
	}

	// Remove hanging volumes
	err = digitaloceanConf.DeleteKubernetesClusterVolumes(resources)
	if err != nil {
		return fmt.Errorf("error deleting kubernetes cluster volumes: %w", err)
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return fmt.Errorf("error creating new GitLab client: %w", err)
		}
		log.Info().Msgf("attempting to delete managed ssh key...")
		err = gitlabClient.DeleteUserSSHKey("kbot-ssh-key")
		if err != nil {
			log.Warn().Msg(err.Error())
		}
	}

	telemetry.SendEvent(telemetryEvent, telemetry.ClusterDeleteCompleted, "")

	cl.Status = constants.ClusterStatusDeleted
	err = secrets.UpdateCluster(kcfg.Clientset, *cl)
	if err != nil {
		return fmt.Errorf("error updating cluster: %w", err)
	}

	err = pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return fmt.Errorf("error resetting K1 directory: %w", err)
	}

	return nil
}
