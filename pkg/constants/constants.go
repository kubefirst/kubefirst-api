package constants

import internal "github.com/kubefirst/kubefirst-api/internal"

const (
	MinimumAvailableDiskSize = internal.MinimumAvailableDiskSize
	MinioDefaultUsername     = internal.MinioDefaultUsername
	MinioDefaultPassword     = internal.MinioDefaultPassword
	KubefirstManifestRepoRef = internal.KubefirstManifestRepoRef
	MinioPortForwardEndpoint = internal.MinioPortForwardEndpoint
	MinioRegion              = internal.MinioRegion
)

var (
	ArgoCDLocalURLTLS           = internal.ArgoCDLocalURLTLS
	KubefirstConsoleLocalURLTLS = internal.KubefirstConsoleLocalURLTLS
)
