package k3d

import internal "github.com/kubefirst/kubefirst-api/internal/k3d"

const (
	DomainName          = internal.DomainName
	GithubHost          = internal.GithubHost
	GitlabHost          = internal.GitlabHost
	K3dVersion          = internal.K3dVersion
	LocalhostOS         = internal.LocalhostOS
	LocalhostARCH       = internal.LocalhostARCH
	CloudProvider       = internal.CloudProvider
	VaultPortForwardURL = internal.VaultPortForwardURL
)

type (
	GitopsDirectoryValues = internal.GitopsDirectoryValues
	MetaphorTokenValues   = internal.MetaphorTokenValues
)

var (
	GetConfig                      = internal.GetConfig
	PrepareGitRepositories         = internal.PrepareGitRepositories
	GetGithubTerraformEnvs         = internal.GetGithubTerraformEnvs
	ClusterCreate                  = internal.ClusterCreate
	GenerateTLSSecrets             = internal.GenerateTLSSecrets
	AddK3DSecrets                  = internal.AddK3DSecrets
	ArgocdURL                      = internal.ArgocdURL
	VaultURL                       = internal.VaultURL
	PostRunPrepareGitopsRepository = internal.PostRunPrepareGitopsRepository
	DownloadTools                  = internal.DownloadTools
	ResolveMinioLocal              = internal.ResolveMinioLocal
	DeleteK3dCluster               = internal.DeleteK3dCluster
	GenerateSingleTLSSecret        = internal.GenerateSingleTLSSecret
)

var ClusterCreateConsoleAPI = internal.ClusterCreateConsoleAPI
