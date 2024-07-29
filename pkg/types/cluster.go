/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package types

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ClusterDefinition describes an incoming request to create a cluster
type ClusterDefinition struct {
	//Cluster
	AdminEmail             string             `json:"admin_email" binding:"required"`
	CloudProvider          string             `json:"cloud_provider" binding:"required,oneof=akamai aws civo digitalocean google k3s vultr"`
	CloudRegion            string             `json:"cloud_region" binding:"required"`
	ClusterName            string             `json:"cluster_name,omitempty"`
	DomainName             string             `json:"domain_name" binding:"required"`
	SubdomainName          string             `json:"subdomain_name,omitempty"`
	DnsProvider            string             `json:"dns_provider,omitempty" binding:"required"`
	Type                   string             `json:"type" binding:"required,oneof=mgmt workload"`
	ForceDestroy           bool               `bson:"force_destroy,omitempty" json:"force_destroy,omitempty"`
	NodeType               string             `json:"node_type" binding:"required"`
	NodeCount              int                `json:"node_count" binding:"required"`
	PostInstallCatalogApps []GitopsCatalogApp `bson:"post_install_catalog_apps,omitempty" json:"post_install_catalog_apps,omitempty"`
	InstallKubefirstPro    bool               `bson:"install_kubefirst_pro,omitempty" json:"install_kubefirst_pro,omitempty"`

	// Git

	// Git
	GitopsTemplateURL    string `json:"gitops_template_url"`
	GitopsTemplateBranch string `json:"gitops_template_branch"`
	GitProvider          string `json:"git_provider" binding:"required,oneof=github gitlab"`
	GitProtocol          string `json:"git_protocol" binding:"required,oneof=ssh https"`
	GitopsRepoName       string `json:"gitopsRepoName" binding:"required"`
	MetaphorRepoName     string `json:"metaphorRepoName" binding:"required"`
	AdminTeamName        string `json:"adminTeamName" binding:"required"`
	DeveloperTeamName    string `json:"developerTeamName" binding:"required"`
	
	// AWS
	ECR bool `json:"ecr,omitempty"`

	//Auth
	AkamaiAuth       AkamaiAuth       `json:"akamai_auth,omitempty"`
	AWSAuth          AWSAuth          `json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `json:"vultr_auth,omitempty"`
	CloudflareAuth   CloudflareAuth   `json:"cloudflare_auth,omitempty"`
	GoogleAuth       GoogleAuth       `json:"google_auth,omitempty"`
	K3sAuth          K3sAuth          `json:"k3s_auth,omitempty"`
	GitAuth          GitAuth          `json:"git_auth,omitempty"`
	LogFileName      string           `bson:"log_file,omitempty" json:"log_file,omitempty"`
}

// Cluster describes the configuration storage for a Kubefirst cluster object
type Cluster struct {
	ID                primitive.ObjectID `bson:"_id" json:"_id"`
	CreationTimestamp string             `bson:"creation_timestamp" json:"creation_timestamp"`

	// Status
	Status        string `bson:"status" json:"status"`
	LastCondition string `bson:"last_condition" json:"last_condition"`
	InProgress    bool   `bson:"in_progress" json:"in_progress"`

	// Identifiers
	AlertsEmail            string             `bson:"alerts_email" json:"alerts_email"`
	CloudProvider          string             `bson:"cloud_provider" json:"cloud_provider"`
	CloudRegion            string             `bson:"cloud_region" json:"cloud_region"`
	ClusterName            string             `bson:"cluster_name" json:"cluster_name"`
	ClusterID              string             `bson:"cluster_id" json:"cluster_id"`
	ClusterType            string             `bson:"cluster_type" json:"cluster_type"`
	DomainName             string             `bson:"domain_name" json:"domain_name"`
	SubdomainName          string             `bson:"subdomain_name" json:"subdomain_name,omitempty"`
	DnsProvider            string             `bson:"dns_provider" json:"dns_provider"`
	PostInstallCatalogApps []GitopsCatalogApp `bson:"post_install_catalog_apps,omitempty" json:"post_install_catalog_apps,omitempty"`

	// Auth
	AkamaiAuth       AkamaiAuth       `bson:"akamai_auth,omitempty" json:"akamai_auth,omitempty"`
	AWSAuth          AWSAuth          `bson:"aws_auth,omitempty" json:"aws_auth,omitempty"`
	CivoAuth         CivoAuth         `bson:"civo_auth,omitempty" json:"civo_auth,omitempty"`
	DigitaloceanAuth DigitaloceanAuth `bson:"do_auth,omitempty" json:"do_auth,omitempty"`
	VultrAuth        VultrAuth        `bson:"vultr_auth,omitempty" json:"vultr_auth,omitempty"`
	CloudflareAuth   CloudflareAuth   `bson:"cloudflare_auth,omitempty" json:"cloudflare_auth,omitempty"`
	GitAuth          GitAuth          `bson:"git_auth,omitempty" json:"git_auth,omitempty"`
	VaultAuth        VaultAuth        `bson:"vault_auth,omitempty" json:"vault_auth,omitempty"`
	GoogleAuth       GoogleAuth       `bson:"google_auth,omitempty" json:"google_auth,omitempty"`
	K3sAuth          K3sAuth          `bson:"k3s_auth,omitempty" json:"k3s_auth,omitempty"`

	GitopsTemplateURL    string `bson:"gitops_template_url" json:"gitops_template_url"`
	GitopsTemplateBranch string `bson:"gitops_template_branch" json:"gitops_template_branch"`
	GitProvider          string `bson:"git_provider" json:"git_provider"`
	GitProtocol          string `bson:"git_protocol" json:"git_protocol"`
	GitopsRepoName       string `bson:"gitopsRepoName" json:"gitopsRepoName"`
	MetaphorRepoName     string `bson:"metaphorRepoName" json:"metaphorRepoName"`
	AdminTeamName        string `bson:"adminTeamName" json:"adminTeamName"`
	DeveloperTeamName    string `bson:"developerTeamName" json:"developerTeamName"`
	GitHost              string `bson:"git_host" json:"git_host"`
	GitlabOwnerGroupID   int    `bson:"gitlab_owner_group_id" json:"gitlab_owner_group_id"`

	AtlantisWebhookSecret string `bson:"atlantis_webhook_secret" json:"atlantis_webhook_secret"`
	AtlantisWebhookURL    string `bson:"atlantis_webhook_url" json:"atlantis_webhook_url"`
	KubefirstTeam         string `bson:"kubefirst_team" json:"kubefirst_team"`
	NodeType              string `bson:"node_type" json:"node_type" binding:"required"`
	NodeCount             int    `bson:"node_count" json:"node_count" binding:"required"`
	LogFileName           string `bson:"log_file,omitempty" json:"log_file,omitempty"`

	StateStoreCredentials StateStoreCredentials `bson:"state_store_credentials,omitempty" json:"state_store_credentials,omitempty"`
	StateStoreDetails     StateStoreDetails     `bson:"state_store_details,omitempty" json:"state_store_details,omitempty"`

	ArgoCDUsername  string `bson:"argocd_username" json:"argocd_username"`
	ArgoCDPassword  string `bson:"argocd_password" json:"argocd_password"`
	ArgoCDAuthToken string `bson:"argocd_auth_token" json:"argocd_auth_token"`

	// Container Registry and Secrets
	ECR bool `bson:"ecr" json:"ecr"`

	// kms
	AWSAccountId              string `bson:"aws_account_id,omitempty" json:"aws_account_id,omitempty"`
	AWSKMSKeyId               string `bson:"aws_kms_key_id,omitempty" json:"aws_kms_key_id,omitempty"`
	AWSKMSKeyDetokenizedCheck bool   `bson:"aws_kms_key_detokenized_check" json:"aws_kms_key_detokenized_check"`

	// Telemetry
	UseTelemetry bool `bson:"use_telemetry"`

	// Checks
	InstallToolsCheck              bool              `bson:"install_tools_check" json:"install_tools_check"`
	DomainLivenessCheck            bool              `bson:"domain_liveness_check" json:"domain_liveness_check"`
	StateStoreCredsCheck           bool              `bson:"state_store_creds_check" json:"state_store_creds_check"`
	StateStoreCreateCheck          bool              `bson:"state_store_create_check" json:"state_store_create_check"`
	GitInitCheck                   bool              `bson:"git_init_check" json:"git_init_check"`
	KbotSetupCheck                 bool              `bson:"kbot_setup_check" json:"kbot_setup_check"`
	GitopsReadyCheck               bool              `bson:"gitops_ready_check" json:"gitops_ready_check"`
	GitTerraformApplyCheck         bool              `bson:"git_terraform_apply_check" json:"git_terraform_apply_check"`
	GitopsPushedCheck              bool              `bson:"gitops_pushed_check" json:"gitops_pushed_check"`
	CloudTerraformApplyCheck       bool              `bson:"cloud_terraform_apply_check" json:"cloud_terraform_apply_check"`
	CloudTerraformApplyFailedCheck bool              `bson:"cloud_terraform_apply_failed_check" json:"cloud_terraform_apply_failed_check"`
	ClusterSecretsCreatedCheck     bool              `bson:"cluster_secrets_created_check" json:"cluster_secrets_created_check"`
	ArgoCDInstallCheck             bool              `bson:"argocd_install_check" json:"argocd_install_check"`
	ArgoCDInitializeCheck          bool              `bson:"argocd_initialize_check" json:"argocd_initialize_check"`
	ArgoCDCreateRegistryCheck      bool              `bson:"argocd_create_registry_check" json:"argocd_create_registry_check"`
	ArgoCDDeleteRegistryCheck      bool              `bson:"argocd_delete_registry_check" json:"argocd_delete_registry_check"`
	VaultInitializedCheck          bool              `bson:"vault_initialized_check" json:"vault_initialized_check"`
	VaultTerraformApplyCheck       bool              `bson:"vault_terraform_apply_check" json:"vault_terraform_apply_check"`
	UsersTerraformApplyCheck       bool              `bson:"users_terraform_apply_check" json:"users_terraform_apply_check"`
	WorkloadClusters               []WorkloadCluster `bson:"workload_clusters,omitempty" json:"workload_clusters,omitempty"`
}

// StateStoreDetails
type StateStoreDetails struct {
	Name                string `bson:"name,omitempty" json:"name,omitempty"`
	ID                  string `bson:"id,omitempty" json:"id,omitempty"`
	Hostname            string `bson:"hostname,omitempty" json:"hostname,omitempty"`
	AWSStateStoreBucket string `bson:"aws_state_store_bucket,omitempty" json:"aws_state_store_bucket,omitempty"`
	AWSArtifactsBucket  string `bson:"aws_artifacts_bucket,omitempty" json:"aws_artifacts_bucket,omitempty"`
}

// PushBucketObject
type PushBucketObject struct {
	LocalFilePath  string `json:"local_file_path"`
	RemoteFilePath string `json:"remote_file_path"`
	ContentType    string `json:"content_type"`
}

type ProxyImportRequest struct {
	Body Cluster `bson:"body" json:"body"`
	Url  string  `bson:"url" json:"url"`
}

type ProxyRequest struct {
	Url string `bson:"url" json:"url"`
}

type Environment struct {
	ID                primitive.ObjectID `bson:"_id" json:"_id"`
	Name              string             `bson:"name" json:"name"`
	Color             string             `bson:"color" json:"color"`
	Description       string             `bson:"description,omitempty" json:"description,omitempty"`
	CreationTimestamp string             `bson:"creation_timestamp" json:"creation_timestamp"`
}

type WorkloadCluster struct {
	AdminEmail        string      `bson:"admin_email,omitempty" json:"admin_email,omitempty"`
	CloudProvider     string      `bson:"cloud_provider,omitempty" json:"cloud_provider,omitempty"`
	ClusterID         string      `bson:"cluster_id,omitempty" json:"cluster_id,omitempty"`
	ClusterName       string      `bson:"cluster_name,omitempty" json:"cluster_name,omitempty"`
	ClusterType       string      `bson:"cluster_type,omitempty" json:"cluster_type,omitempty"`
	CloudRegion       string      `bson:"cloud_region,omitempty" json:"cloud_region,omitempty"`
	CreationTimestamp string      `bson:"creation_timestamp" json:"creation_timestamp"`
	DomainName        string      `bson:"domain_name,omitempty" json:"domain_name,omitempty"`
	DnsProvider       string      `bson:"dns_provider,omitempty" json:"dns_provider,omitempty"`
	Environment       Environment `bson:"environment,omitempty" json:"environment,omitempty"`
	GitAuth           GitAuth     `bson:"git_auth,omitempty" json:"git_auth,omitempty"`
	InstanceSize      string      `bson:"instance_size,omitempty" json:"instance_size,omitempty"`
	NodeType          string      `bson:"node_type,omitempty" json:"node_type,omitempty"`
	NodeCount         int         `bson:"node_count,omitempty" json:"node_count,omitempty"`
	Status            string      `bson:"status,omitempty" json:"status,omitempty"`
	GitopsRepoName    string      `bson:"gitops_repo_name,omitempty" json:"gitops_repo_name,omitempty"`
}

type WorkloadClusterSet struct {
	Clusters []WorkloadCluster `json:"clusters"`
}
