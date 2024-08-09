/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	"github.com/kubefirst/kubefirst-api/pkg/google"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
)

// googleHandoffScreen prints the handoff screen
func GoogleHandoffScreen(clusterName, domainName string, gitOwner string, config *providerConfigs.ProviderConfig, silentMode bool) {
	renderHandoff(Opts{
		ClusterName:             clusterName,
		DomainName:              domainName,
		GitOwner:                gitOwner,
		GitProvider:             config.GitProvider,
		DestinationGitopsRepo:   config.DestinationGitopsRepoHTTPSURL,
		DestinationMetaphorRepo: config.DestinationMetaphorRepoHTTPSURL,
		CloudProvider:           google.CloudProvider,
	}, silentMode)
}
