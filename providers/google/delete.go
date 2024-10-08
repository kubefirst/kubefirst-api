/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package google

import (
	"context"
	"fmt"
	"strconv"
	"time"

	googleext "github.com/konstructio/kubefirst-api/extensions/google"
	terraformext "github.com/konstructio/kubefirst-api/extensions/terraform"
	pkg "github.com/konstructio/kubefirst-api/internal"
	"github.com/konstructio/kubefirst-api/internal/argocd"
	"github.com/konstructio/kubefirst-api/internal/constants"
	"github.com/konstructio/kubefirst-api/internal/errors"
	gitlab "github.com/konstructio/kubefirst-api/internal/gitlab"
	"github.com/konstructio/kubefirst-api/internal/httpCommon"
	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/internal/secrets"
	"github.com/konstructio/kubefirst-api/internal/utils"
	"github.com/konstructio/kubefirst-api/pkg/google"
	"github.com/konstructio/kubefirst-api/pkg/providerConfigs"
	pkgtypes "github.com/konstructio/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// DeleteGoogleCluster
func DeleteGoogleCluster(cl *pkgtypes.Cluster, telemetryEvent telemetry.TelemetryEvent) error {
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
		return fmt.Errorf("error updating cluster status: %w", err)
	}

	switch cl.GitProvider {
	case "github":
		if cl.GitTerraformApplyCheck {
			log.Info().Msg("destroying github resources with terraform")

			tfEntrypoint := config.GitopsDir + "/terraform/github"
			tfEnvs := map[string]string{}
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
			tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, cl)
			err := terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
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
				return fmt.Errorf("error creating gitlab client: %w", err)
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
			tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
			tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, gitlabClient.ParentGroupID, cl)
			err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
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

	if cl.CloudTerraformApplyCheck || cl.CloudTerraformApplyFailedCheck {
		if !cl.ArgoCDDeleteRegistryCheck {
			googleConf := google.Configuration{
				Context: context.Background(),
				Project: cl.GoogleAuth.ProjectID,
				Region:  cl.CloudRegion,
			}
			kcfg, _ := googleConf.GetContainerClusterAuth(cl.ClusterName, []byte(cl.GoogleAuth.KeyFile))

			log.Info().Msg("destroying google resources with terraform")

			// Only port-forward to ArgoCD and delete registry if ArgoCD was installed
			if cl.ArgoCDInstallCheck {
				removeArgoCDApps := []string{"ingress-nginx-components", "ingress-nginx"}
				err = argocd.ApplicationCleanup(kcfg.Clientset, removeArgoCDApps)
				if err != nil {
					log.Error().Msgf("encountered error during argocd application cleanup: %s", err)
				}

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
					return fmt.Errorf("error reading argocd secret: %w", err)
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
					return fmt.Errorf("error deleting registry application: %w", err)
				}
				log.Info().Msgf("http status code %d", httpCode)
			}

			// Pause before cluster destroy to prevent a race condition
			log.Info().Msg("waiting for google Kubernetes cluster resource removal to finish...")
			time.Sleep(time.Second * 10)

			cl.ArgoCDDeleteRegistryCheck = true
			err = secrets.UpdateCluster(kcfg.Clientset, *cl)
			if err != nil {
				return fmt.Errorf("error updating cluster: %w", err)
			}
		}

		log.Info().Msg("destroying google cloud resources")
		tfEntrypoint := config.GitopsDir + fmt.Sprintf("/terraform/%s", cl.CloudProvider)
		tfEnvs := map[string]string{}
		tfEnvs = googleext.GetGoogleTerraformEnvs(tfEnvs, cl)
		tfEnvs["TF_VAR_project"] = cl.GoogleAuth.ProjectID

		switch cl.GitProvider {
		case "github":
			tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, cl)
		case "gitlab":
			gid, err := strconv.Atoi(strconv.Itoa(cl.GitlabOwnerGroupID))
			if err != nil {
				return fmt.Errorf("couldn't convert gitlab group id to int: %w", err)
			}
			tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, gid, cl)
		}
		err = terraformext.InitDestroyAutoApprove(config.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error executing terraform destroy %s", tfEntrypoint)
			errors.HandleClusterError(cl, err.Error())
			return fmt.Errorf("error executing terraform destroy %s: %w", tfEntrypoint, err)
		}
		log.Info().Msg("google resources terraform destroyed")

		cl.CloudTerraformApplyCheck = false
		cl.CloudTerraformApplyFailedCheck = false
		err = secrets.UpdateCluster(kcfg.Clientset, *cl)
		if err != nil {
			return fmt.Errorf("error updating cluster status: %w", err)
		}
	}

	// remove ssh key provided one was created
	if cl.GitProvider == "gitlab" {
		gitlabClient, err := gitlab.NewGitLabClient(cl.GitAuth.Token, cl.GitAuth.Owner)
		if err != nil {
			return fmt.Errorf("error creating gitlab client: %w", err)
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
		return fmt.Errorf("error updating cluster status: %w", err)
	}

	err = pkg.ResetK1Dir(config.K1Dir)
	if err != nil {
		return fmt.Errorf("error resetting k1 directory: %w", err)
	}

	return nil
}
