/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package reports

import (
	awsinternal "github.com/kubefirst/kubefirst-api/internal/aws"
	"github.com/kubefirst/kubefirst-api/pkg/providerConfigs"
)

// AwsHandoffScreen prints the handoff screen
func AwsHandoffScreen(clusterName, domainName string, gitOwner string, config *providerConfigs.ProviderConfig, silentMode bool) {
	renderHandoff(Opts{
		ClusterName:             clusterName,
		DomainName:              domainName,
		GitOwner:                gitOwner,
		GitProvider:             config.GitProvider,
		DestinationGitopsRepo:   config.DestinationGitopsRepoHTTPSURL,
		DestinationMetaphorRepo: config.DestinationMetaphorRepoHTTPSURL,
		CloudProvider:           awsinternal.CloudProvider,
	}, silentMode)
}
