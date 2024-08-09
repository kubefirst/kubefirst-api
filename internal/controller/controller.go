/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package controller

import (
	"context"
	"fmt"
	"net/http"
	"time"

	runtime "github.com/kubefirst/kubefirst-api/internal"
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/github"
	"github.com/kubefirst/kubefirst-api/internal/gitlab"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/services"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	google "github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/handlers"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/kubefirst/metrics-client/pkg/telemetry"
	"k8s.io/client-go/kubernetes"

	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ClusterController struct {
	CloudProvider             string
	CloudRegion               string
	ClusterName               string
	ClusterID                 string
	ClusterType               string
	DomainName                string
	SubdomainName             string
	DNSProvider               string
	UseCloudflareOriginIssuer bool
	AlertsEmail               string

	// auth
	AkamaiAuth             types.AkamaiAuth
	AWSAuth                types.AWSAuth
	CivoAuth               types.CivoAuth
	DigitaloceanAuth       types.DigitaloceanAuth
	VultrAuth              types.VultrAuth
	CloudflareAuth         types.CloudflareAuth
	GitAuth                types.GitAuth
	VaultAuth              types.VaultAuth
	GoogleAuth             types.GoogleAuth
	K3sAuth                types.K3sAuth
	AwsAccessKeyID         string
	AwsSecretAccessKey     string
	NodeType               string
	NodeCount              int
	PostInstallCatalogApps []types.GitopsCatalogApp
	InstallKubefirstPro    bool

	// configs
	ProviderConfig providerConfigs.ProviderConfig

	// git
	GitopsTemplateURL    string
	GitopsTemplateBranch string
	GitProvider          string
	GitProtocol          string
	GitHost              string
	GitOwner             string
	GitUser              string
	GitToken             string
	GitlabOwnerGroupID   int

	// container registry
	ContainerRegistryHost string
	ECR                   bool

	// http
	Client *http.Client

	// repositories
	Repositories []string

	// teams
	Teams []string

	// atlantis
	AtlantisWebhookSecret string
	AtlantisWebhookURL    string

	// internal
	KubefirstTeam string

	// state store
	KubefirstStateStoreBucketName string
	KubefirstArtifactsBucketName  string

	KubernetesClient *kubernetes.Clientset

	// Telemetry
	TelemetryEvent telemetry.TelemetryEvent

	// Provider clients
	AwsClient    *awsinternal.Configuration
	GoogleClient google.Configuration
	Kcfg         *k8s.KubernetesClient
	Cluster      types.Cluster
}

// InitController
func (clctrl *ClusterController) InitController(def *types.ClusterDefinition) error {
	// Create k1 dir if it doesn't exist
	utils.CreateK1Directory(def.ClusterName)

	// Get Environment variables
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	kcfg := utils.GetKubernetesClient(def.ClusterName)
	clctrl.KubernetesClient = kcfg.Clientset

	// Determine if record already exists
	recordExists := true
	rec, err := secrets.GetCluster(clctrl.KubernetesClient, def.ClusterName)
	if rec.ClusterID == "" && err != nil {
		recordExists = false
		log.Info().Msg("cluster record doesn't exist, continuing")
	}

	logFileName := def.LogFileName
	if recordExists {
		logFileName = rec.LogFileName
	}

	utils.InitializeLogs(logFileName)

	// If record exists but status is deleted, entry should be deleted
	// and process should start fresh
	if recordExists && rec.Status == constants.ClusterStatusDeleted {
		err = secrets.DeleteCluster(clctrl.KubernetesClient, def.ClusterName)
		if err != nil {
			return fmt.Errorf("could not delete existing cluster %s: %s", def.ClusterName, err)
		}
	}

	var clusterID string
	if recordExists {
		clusterID = rec.ClusterID
	} else {
		clusterID = runtime.GenerateClusterID()
	}

	telemetryEvent := telemetry.TelemetryEvent{
		CliVersion:        env.KubefirstVersion,
		CloudProvider:     env.CloudProvider,
		ClusterID:         env.ClusterID,
		ClusterType:       env.ClusterType,
		DomainName:        env.DomainName,
		ErrorMessage:      "",
		GitProvider:       env.GitProvider,
		InstallMethod:     env.InstallMethod,
		KubefirstClient:   "api",
		KubefirstTeam:     env.KubefirstTeam,
		KubefirstTeamInfo: env.KubefirstTeamInfo,
		MachineID:         env.ClusterID,
		ParentClusterId:   env.ParentClusterID,
		MetricName:        telemetry.ClusterInstallCompleted,
		UserId:            env.ClusterID,
	}
	clctrl.TelemetryEvent = telemetryEvent

	// Copy Cluster Definiion to Cluster Controller
	clctrl.AlertsEmail = def.AdminEmail
	clctrl.CloudProvider = def.CloudProvider
	clctrl.CloudRegion = def.CloudRegion
	clctrl.ClusterName = def.ClusterName
	clctrl.ClusterID = clusterID
	clctrl.DomainName = def.DomainName
	clctrl.SubdomainName = def.SubdomainName
	clctrl.DNSProvider = def.DNSProvider
	clctrl.ClusterType = def.Type
	clctrl.Client = http.DefaultClient
	clctrl.NodeType = def.NodeType
	clctrl.NodeCount = def.NodeCount
	clctrl.PostInstallCatalogApps = def.PostInstallCatalogApps
	clctrl.InstallKubefirstPro = def.InstallKubefirstPro

	clctrl.AkamaiAuth = def.AkamaiAuth
	clctrl.AWSAuth = def.AWSAuth
	clctrl.CivoAuth = def.CivoAuth
	clctrl.DigitaloceanAuth = def.DigitaloceanAuth
	clctrl.VultrAuth = def.VultrAuth
	clctrl.GoogleAuth = def.GoogleAuth
	clctrl.K3sAuth = def.K3sAuth
	clctrl.CloudflareAuth = def.CloudflareAuth

	clctrl.Repositories = []string{"gitops", "metaphor"}
	clctrl.Teams = []string{"admins", "developers"}

	clctrl.ECR = def.ECR

	if def.GitopsTemplateBranch != "" {
		clctrl.GitopsTemplateBranch = def.GitopsTemplateBranch
	} else {
		clctrl.GitopsTemplateBranch = env.KubefirstVersion
	}

	if def.GitopsTemplateURL != "" {
		if def.GitopsTemplateBranch != "" {
			clctrl.GitopsTemplateURL = def.GitopsTemplateURL
		} else {
			return fmt.Errorf("must supply branch of gitops templatelog.Fatal().Msg( repo when supplying a gitops template url")
		}
	} else {
		clctrl.GitopsTemplateURL = "https://github.com/kubefirst/gitops-template.git"
	}
	if def.CloudProvider == "akamai" {
		clctrl.KubefirstStateStoreBucketName = clctrl.ClusterName
	} else {
		clctrl.KubefirstStateStoreBucketName = fmt.Sprintf("k1-state-store-%s-%s", clctrl.ClusterName, clusterID)
	}

	clctrl.KubefirstArtifactsBucketName = fmt.Sprintf("k1-artifacts-%s-%s", clctrl.ClusterName, clusterID)
	clctrl.NodeType = def.NodeType
	clctrl.NodeCount = def.NodeCount

	clctrl.KubefirstTeam = env.KubefirstTeam

	clctrl.AtlantisWebhookSecret = runtime.Random(20)

	var fullDomainName string
	if clctrl.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", clctrl.SubdomainName, clctrl.DomainName)
	} else {
		fullDomainName = clctrl.DomainName
	}
	clctrl.AtlantisWebhookURL = fmt.Sprintf("https://atlantis.%s/events", fullDomainName)

	// Initialize git parameters
	clctrl.GitProvider = def.GitProvider
	clctrl.GitProtocol = def.GitProtocol
	clctrl.GitAuth = def.GitAuth

	err = clctrl.SetGitTokens(*def)
	if err != nil {
		return err
	}

	// Instantiate provider configuration
	switch clctrl.CloudProvider {
	case "akamai":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.AkamaiToken = clctrl.AkamaiAuth.Token
	case "aws":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
	case "civo":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.APIToken, clctrl.CloudflareAuth.OriginCaIssuerKey)
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.CivoToken = clctrl.CivoAuth.Token
	case "google":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.GoogleAuth = clctrl.GoogleAuth.KeyFile
		clctrl.ProviderConfig.GoogleProject = clctrl.GoogleAuth.ProjectID
	case "digitalocean":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.DigitaloceanToken = clctrl.DigitaloceanAuth.Token
	case "vultr":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.VultrToken = clctrl.VultrAuth.Token
	case "k3s":
		conf, err := providerConfigs.GetConfig(clctrl.ClusterName, clctrl.DomainName, clctrl.GitProvider, clctrl.GitAuth.Owner, clctrl.GitProtocol, clctrl.CloudflareAuth.Token, "")
		if err != nil {
			return fmt.Errorf("unable to get configuration: %w", err)
		}

		clctrl.ProviderConfig = *conf
		clctrl.ProviderConfig.K3sServersPrivateIps = clctrl.K3sAuth.K3sServersPrivateIps
		clctrl.ProviderConfig.K3sServersPublicIps = clctrl.K3sAuth.K3sServersPublicIps
		clctrl.ProviderConfig.K3sSSHPrivateKey = clctrl.K3sAuth.K3sSSHPrivateKey
		clctrl.ProviderConfig.K3sSSHUser = clctrl.K3sAuth.K3sSSHUser
		clctrl.ProviderConfig.K3sServersArgs = clctrl.K3sAuth.K3sServersArgs
	}

	// Instantiate provider clients and copy cluster controller to cluster type
	switch clctrl.CloudProvider {
	case "aws":
		conf, err := awsinternal.NewAwsV3(
			clctrl.CloudRegion,
			clctrl.AWSAuth.AccessKeyID,
			clctrl.AWSAuth.SecretAccessKey,
			clctrl.AWSAuth.SessionToken,
		)
		if err != nil {
			return fmt.Errorf("unable to create AWS client: %w", err)
		}

		clctrl.AwsClient = &awsinternal.Configuration{Config: conf}
	case "google":
		clctrl.GoogleClient = google.Configuration{
			Context: context.Background(),
			Project: def.GoogleAuth.ProjectID,
			Region:  clctrl.CloudRegion,
		}
	}

	// Write cluster record if it doesn't exist
	clctrl.Cluster = types.Cluster{
		ID:                     primitive.NewObjectID(),
		CreationTimestamp:      fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC())),
		Status:                 constants.ClusterStatusProvisioning,
		AlertsEmail:            clctrl.AlertsEmail,
		ClusterName:            clctrl.ClusterName,
		CloudProvider:          clctrl.CloudProvider,
		CloudRegion:            clctrl.CloudRegion,
		DomainName:             clctrl.DomainName,
		SubdomainName:          clctrl.SubdomainName,
		DNSProvider:            clctrl.DNSProvider,
		ClusterID:              clctrl.ClusterID,
		ECR:                    clctrl.ECR,
		ClusterType:            clctrl.ClusterType,
		GitopsTemplateURL:      clctrl.GitopsTemplateURL,
		GitopsTemplateBranch:   clctrl.GitopsTemplateBranch,
		GitProvider:            clctrl.GitProvider,
		GitProtocol:            clctrl.GitProtocol,
		GitHost:                clctrl.GitHost,
		GitAuth:                clctrl.GitAuth,
		GitlabOwnerGroupID:     clctrl.GitlabOwnerGroupID,
		AtlantisWebhookSecret:  clctrl.AtlantisWebhookSecret,
		AtlantisWebhookURL:     clctrl.AtlantisWebhookURL,
		KubefirstTeam:          clctrl.KubefirstTeam,
		AkamaiAuth:             clctrl.AkamaiAuth,
		AWSAuth:                clctrl.AWSAuth,
		CivoAuth:               clctrl.CivoAuth,
		GoogleAuth:             clctrl.GoogleAuth,
		DigitaloceanAuth:       clctrl.DigitaloceanAuth,
		VultrAuth:              clctrl.VultrAuth,
		K3sAuth:                clctrl.K3sAuth,
		CloudflareAuth:         clctrl.CloudflareAuth,
		NodeType:               clctrl.NodeType,
		NodeCount:              clctrl.NodeCount,
		LogFileName:            def.LogFileName,
		PostInstallCatalogApps: clctrl.PostInstallCatalogApps,
	}

	if !recordExists {
		if env.K1LocalDebug == "true" {
			err = utils.CreateKubefirstNamespace(clctrl.KubernetesClient)
			if err != nil {
				return err
			}
		}

		err = secrets.InsertCluster(clctrl.KubernetesClient, clctrl.Cluster)
		if err != nil {
			return err
		}
	} else {
		clctrl.Cluster = rec
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) SetGitTokens(def types.ClusterDefinition) error {
	switch def.GitProvider {
	case "github":
		gitHubService := services.NewGitHubService(clctrl.Client)
		gitHubHandler := handlers.NewGitHubHandler(gitHubService)

		clctrl.GitHost = "github.com"
		clctrl.ContainerRegistryHost = "ghcr.io"
		// Verify token scopes
		err := github.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return err
		}
		// Get authenticated user's name
		githubUser, err := gitHubHandler.GetGitHubUser(def.GitAuth.Token)
		if err != nil {
			return err
		}
		clctrl.GitAuth.User = githubUser
	case "gitlab":
		clctrl.GitHost = "gitlab.com"
		clctrl.ContainerRegistryHost = "registry.gitlab.com"
		// Verify token scopes
		err := gitlab.VerifyTokenPermissions(def.GitAuth.Token)
		if err != nil {
			return err
		}
		gitlabClient, err := gitlab.NewGitLabClient(def.GitAuth.Token, def.GitAuth.Owner)
		if err != nil {
			return err
		}
		clctrl.GitAuth.Owner = gitlabClient.ParentGroupPath
		clctrl.GitlabOwnerGroupID = gitlabClient.ParentGroupID
		// Get authenticated user's name
		user, _, err := gitlabClient.Client.Users.CurrentUser()
		if err != nil {
			return fmt.Errorf("unable to get authenticated user info - please make sure GITLAB_TOKEN env var is set %s", err.Error())
		}
		clctrl.GitAuth.User = user.Username
	default:
		return fmt.Errorf("invalid git provider option")
	}

	return nil
}

// GetCurrentClusterRecord will return an active cluster's record if it exists
func (clctrl *ClusterController) GetCurrentClusterRecord() (types.Cluster, error) {
	cl, err := secrets.GetCluster(clctrl.KubernetesClient, clctrl.ClusterName)
	if err != nil {
		return types.Cluster{}, err
	}

	return cl, nil
}

// UpdateClusterOnError implements an error handler for cluster controller objects
func (clctrl *ClusterController) UpdateClusterOnError(condition string) error {
	clctrl.Cluster.InProgress = false
	clctrl.Cluster.Status = constants.ClusterStatusError
	clctrl.Cluster.LastCondition = condition

	log.Error().Msgf("unexpected error: %s", condition)
	if err := secrets.UpdateCluster(clctrl.KubernetesClient, clctrl.Cluster); err != nil {
		return err
	}

	return nil
}
