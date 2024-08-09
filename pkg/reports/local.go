/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	"github.com/kubefirst/kubefirst-api/internal/k3d"
)

// LocalHandoffScreenV2 prints the handoff screen
func LocalHandoffScreenV2(clusterName, gitDestDescriptor string, gitOwner string, config *k3d.Config, silentMode bool) {
	renderHandoff(Opts{
		ClusterName:             clusterName,
		DomainName:              k3d.DomainName,
		GitOwner:                gitOwner,
		GitProvider:             config.GitProvider,
		DestinationGitopsRepo:   config.DestinationGitopsRepoURL,
		DestinationMetaphorRepo: config.DestinationMetaphorRepoURL,
		CloudProvider:           k3d.CloudProvider,
		MkCertClient:            config.MkCertClient,
		CustomOwnerName:         gitDestDescriptor,
	}, silentMode)
}
