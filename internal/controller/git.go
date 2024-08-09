/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"fmt"
	"time"

	akamaiext "github.com/kubefirst/kubefirst-api/extensions/akamai"
	awsext "github.com/kubefirst/kubefirst-api/extensions/aws"
	civoext "github.com/kubefirst/kubefirst-api/extensions/civo"
	digitaloceanext "github.com/kubefirst/kubefirst-api/extensions/digitalocean"
	googleext "github.com/kubefirst/kubefirst-api/extensions/google"
	k3sext "github.com/kubefirst/kubefirst-api/extensions/k3s"
	terraformext "github.com/kubefirst/kubefirst-api/extensions/terraform"
	vultrext "github.com/kubefirst/kubefirst-api/extensions/vultr"
	gitShim "github.com/kubefirst/kubefirst-api/internal/gitShim"
	"github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	log "github.com/rs/zerolog/log"
)

// GitInit
func (clctrl *ClusterController) GitInit() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	if !cl.GitInitCheck {
		// Check for git resources in provider
		initGitParameters := gitShim.GitInitParameters{
			GitProvider:  clctrl.GitProvider,
			GitToken:     clctrl.GitAuth.Token,
			GitOwner:     clctrl.GitAuth.Owner,
			GitProtocol:  clctrl.GitProtocol,
			Repositories: clctrl.Repositories,
			Teams:        clctrl.Teams,
			GithubOrg:    clctrl.GitAuth.Owner,
			GitlabGroup:  clctrl.GitAuth.Owner,
		}
		err := gitShim.InitializeGitProvider(&initGitParameters)
		if err != nil {
			return err
		}

		clctrl.Cluster.GitInitCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

// RunGitTerraform
func (clctrl *ClusterController) RunGitTerraform() error {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return err
	}

	// // * create teams and repositories in github

	telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitTerraformApplyStarted, "")

	log.Info().Msgf("Creating %s resources with terraform", clctrl.GitProvider)

	tfEntrypoint := clctrl.ProviderConfig.GitopsDir + fmt.Sprintf("/terraform/%s", clctrl.GitProvider)
	tfEnvs := map[string]string{}

	if !cl.GitTerraformApplyCheck {
		switch clctrl.GitProvider {
		case "github":
			switch clctrl.CloudProvider {
			case "akamai":
				tfEnvs = akamaiext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "aws":
				tfEnvs = awsext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "civo":
				tfEnvs = civoext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "google":
				tfEnvs = googleext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "digitalocean":
				tfEnvs = digitaloceanext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "vultr":
				tfEnvs = vultrext.GetGithubTerraformEnvs(tfEnvs, &cl)
			case "k3s":
				tfEnvs = k3sext.GetGithubTerraformEnvs(tfEnvs, &cl)
			}
		case "gitlab":
			switch clctrl.CloudProvider {
			case "akamai":
				tfEnvs = akamaiext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "aws":
				tfEnvs = awsext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "civo":
				tfEnvs = civoext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "google":
				tfEnvs = googleext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "digitalocean":
				tfEnvs = digitaloceanext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "vultr":
				tfEnvs = vultrext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			case "k3s":
				tfEnvs = k3sext.GetGitlabTerraformEnvs(tfEnvs, clctrl.GitlabOwnerGroupID, &cl)
			}
		}

		err := terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
		if err != nil {
			log.Error().Msgf("error applying git terraform: %s", err)
			log.Info().Msg("sleeping 10 seconds before retrying terraform execution once more")
			time.Sleep(10 * time.Second)
			err = terraformext.InitApplyAutoApprove(clctrl.ProviderConfig.TerraformClient, tfEntrypoint, tfEnvs)
			if err != nil {
				msg := fmt.Sprintf("error creating %s resources with terraform %s: %s", clctrl.GitProvider, tfEntrypoint, err)
				log.Error().Msg(msg)
				telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitTerraformApplyFailed, err.Error())
				return fmt.Errorf(msg)
			}
		}

		log.Info().Msgf("created git projects and groups for %s.com/%s", clctrl.GitProvider, clctrl.GitAuth.Owner)
		telemetry.SendEvent(clctrl.TelemetryEvent, telemetry.GitTerraformApplyCompleted, "")

		clctrl.Cluster.GitTerraformApplyCheck = true
		err = secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	}

	return nil
}

func (clctrl *ClusterController) GetRepoURL() (string, error) {
	// default case is https
	destinationGitopsRepoURL := clctrl.ProviderConfig.DestinationGitopsRepoURL

	switch clctrl.GitProvider {
	case "github":

		// Define constant url based on flag input, only expecting 2 protocols
		if clctrl.ProviderConfig.GitProtocol == "ssh" {
			destinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
		}
	case "gitlab":
		gitlabClient, err := gitlab.NewGitLabClient(clctrl.GitAuth.Token, clctrl.GitAuth.Owner)
		if err != nil {
			return "", err
		}
		// Format git url based on full path to group
		switch clctrl.ProviderConfig.GitProtocol {
		case "https":
			// Update the urls in the cluster for gitlab parent groups
			clctrl.ProviderConfig.DestinationGitopsRepoHTTPSURL = fmt.Sprintf("https://gitlab.com/%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoHTTPSURL = fmt.Sprintf("https://gitlab.com/%s/metaphor.git", gitlabClient.ParentGroupPath)
		default:
			// Update the urls in the cluster for gitlab parent group
			clctrl.ProviderConfig.DestinationGitopsRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/gitops.git", gitlabClient.ParentGroupPath)
			clctrl.ProviderConfig.DestinationMetaphorRepoGitURL = fmt.Sprintf("git@gitlab.com:%s/metaphor.git", gitlabClient.ParentGroupPath)
			// Return the url used for detokenization
			destinationGitopsRepoURL = clctrl.ProviderConfig.DestinationGitopsRepoGitURL
		}
	}

	return destinationGitopsRepoURL, nil
}
