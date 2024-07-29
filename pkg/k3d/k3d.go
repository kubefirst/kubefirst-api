package k3d

import internal "github.com/kubefirst/kubefirst-api/internal/k3d"

const DomainName = internal.DomainName
const GithubHost = internal.GithubHost
const GitlabHost = internal.GitlabHost
const K3dVersion = internal.K3dVersion
const LocalhostOS = internal.LocalhostOS
const LocalhostARCH = internal.LocalhostARCH
const CloudProvider = internal.CloudProvider
const VaultPortForwardURL = internal.VaultPortForwardURL

type GitopsDirectoryValues = internal.GitopsDirectoryValues
type MetaphorTokenValues = internal.MetaphorTokenValues

var GetConfig = internal.GetConfig
var PrepareGitRepositories = internal.PrepareGitRepositories
var GetGithubTerraformEnvs = internal.GetGithubTerraformEnvs
var ClusterCreate = internal.ClusterCreate
var GenerateTLSSecrets = internal.GenerateTLSSecrets
var AddK3DSecrets = internal.AddK3DSecrets
var ArgocdURL = internal.ArgocdURL
var VaultURL = internal.VaultURL
var PostRunPrepareGitopsRepository = internal.PostRunPrepareGitopsRepository
var DownloadTools = internal.DownloadTools
var ResolveMinioLocal = internal.ResolveMinioLocal
var DeleteK3dCluster = internal.DeleteK3dCluster
var GenerateSingleTLSSecret = internal.GenerateSingleTLSSecret

var ClusterCreateConsoleAPI = internal.ClusterCreateConsoleAPI
var TerraformPrep = internal.TerraformPrep