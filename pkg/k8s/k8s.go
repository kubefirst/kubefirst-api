package k8s

import internal "github.com/kubefirst/kubefirst-api/internal/k8s"

type KubernetesClient = internal.KubernetesClient

var CheckForExistingPortForwards = internal.CheckForExistingPortForwards

var (
	CreateKubeConfig          = internal.CreateKubeConfig
	ReturnDeploymentObject    = internal.ReturnDeploymentObject
	WaitForDeploymentReady    = internal.WaitForDeploymentReady
	VerifyArgoCDReadiness     = internal.VerifyArgoCDReadiness
	GetSecretValue            = internal.GetSecretValue
	OpenPortForwardPodWrapper = internal.OpenPortForwardPodWrapper
	ReturnStatefulSetObject   = internal.ReturnStatefulSetObject
	WaitForStatefulSetReady   = internal.WaitForStatefulSetReady
	ReturnJobObject           = internal.ReturnJobObject
	WaitForJobComplete        = internal.WaitForJobComplete
	UpdateSecretV2            = internal.UpdateSecretV2
)

var (
	ReadSecretV2Old = internal.ReadSecretV2Old
	ReadSecretV2    = internal.ReadSecretV2
	ReadService     = internal.ReadService
	CreateSecretV2  = internal.CreateSecretV2
)
