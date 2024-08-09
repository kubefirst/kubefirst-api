package argocd

import "github.com/kubefirst/kubefirst-api/internal/argocd"

//nolint:gochecknoglobals
var (
	ArgocdSecretClient         = argocd.ArgocdSecretClient
	GetArgocdTokenV2           = argocd.GetArgocdTokenV2
	GetArgoCDApplicationObject = argocd.GetArgoCDApplicationObject
	RefreshApplication         = argocd.RefreshApplication
	RefreshRegistryApplication = argocd.RefreshRegistryApplication
)
