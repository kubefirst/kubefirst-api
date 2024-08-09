/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package environments

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/httpCommon"
	"github.com/kubefirst/kubefirst-api/internal/secrets"
	"github.com/kubefirst/kubefirst-api/internal/utils"
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewEnvironment(envDef types.Environment) (types.Environment, error) {
	// Create new environment
	envDef.CreationTimestamp = fmt.Sprintf("%v", primitive.NewDateTimeFromTime(time.Now().UTC()))

	kcfg := utils.GetKubernetesClient("TODO: Secrets")
	newEnv, err := secrets.InsertEnvironment(kcfg.Clientset, envDef)

	return newEnv, err
}

func CreateDefaultClusters(mgmtCluster types.Cluster) error {
	defaultClusterNames := []string{"development", "staging", "production"}

	defaultVclusterTemplate := types.WorkloadCluster{
		AdminEmail:    mgmtCluster.AlertsEmail,
		CloudProvider: mgmtCluster.CloudProvider,
		ClusterID:     mgmtCluster.ClusterID,
		ClusterName:   "not so empty string which should be replaced",
		ClusterType:   "workload-vcluster",
		CloudRegion:   mgmtCluster.CloudRegion,
		DomainName:    "not so empty string which should be replaced",
		DNSProvider:   mgmtCluster.DNSProvider,
		Environment: types.Environment{
			Name:        "not so empty string which should be replaced",
			Description: "not so empty string which should be replaced",
		},
		GitAuth:      mgmtCluster.GitAuth,
		InstanceSize: "", // left up to terraform
		NodeType:     "", // left up to terraform
		NodeCount:    3,  // defaulted here
	}

	defaultClusters := []types.WorkloadCluster{}
	kcfg := utils.GetKubernetesClient("TODO: Secrets")
	secrets.CreateSecretReference(kcfg.Clientset, secrets.KubefirstEnvironmentSecretName, types.SecretListReference{
		Name: "environments",
	})

	for _, clusterName := range defaultClusterNames {
		vcluster := defaultVclusterTemplate
		vcluster.ClusterName = clusterName
		vcluster.Environment.Name = clusterName
		vcluster.DomainName = fmt.Sprintf("%s.%s", clusterName, mgmtCluster.DomainName)
		vcluster.Environment.Description = fmt.Sprintf("Default %s environment", clusterName)
		switch clusterName {
		case "development":
			vcluster.Environment.Color = "green"
		case "staging":
			vcluster.Environment.Color = "gold"
		case "production":
			vcluster.Environment.Color = "pink"
		}

		var err error
		vcluster.Environment, err = NewEnvironment(vcluster.Environment)
		if err != nil {
			log.Error().Msgf("error creating default environment in db for env %s", err)
			return fmt.Errorf("error creating default environment in db for environment %q: %w", clusterName, err)
		}
		defaultClusters = append(defaultClusters, vcluster)
	}

	defaultEnvironmentSet := types.WorkloadClusterSet{
		Clusters: defaultClusters,
	}

	var fullDomainName string
	if mgmtCluster.SubdomainName != "" {
		fullDomainName = fmt.Sprintf("%s.%s", mgmtCluster.SubdomainName, mgmtCluster.DomainName)
	} else {
		fullDomainName = mgmtCluster.DomainName
	}

	for _, clusterName := range defaultClusterNames {
		// Add to list
		err := secrets.CreateClusterServiceList(kcfg.Clientset, clusterName)
		if err != nil {
			return fmt.Errorf("error creating cluster service list for cluster %q: %w", clusterName, err)
		}

		// Update list
		err = secrets.InsertClusterServiceListEntry(kcfg.Clientset, clusterName, &types.Service{
			Name:        "Metaphor",
			Default:     true,
			Description: "A multi-environment demonstration space for frontend application best practices that's easy to apply to other projects.",
			Image:       "https://assets.kubefirst.com/console/metaphor.svg",
			Links:       []string{fmt.Sprintf("https://metaphor-%s.%s", clusterName, fullDomainName)},
			Status:      "",
		})
		if err != nil {
			return fmt.Errorf("error inserting cluster service list entry for cluster %q: %w", clusterName, err)
		}
	}

	// call api-ee to create clusters
	return callAPIEE(defaultEnvironmentSet)
}

func callAPIEE(payload types.WorkloadClusterSet) error {
	httpClient := httpCommon.CustomHTTPClient(false)
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	for i, cluster := range payload.Clusters {
		log.Info().Msgf("creating cluster %s for %s", strconv.Itoa(i), cluster.ClusterName)

		payload, err := json.Marshal(cluster)
		if err != nil {
			return fmt.Errorf("error marshalling cluster %q: %w", cluster.ClusterName, err)
		}

		endpoint := fmt.Sprintf("%s/api/v1/cluster/%s", env.EnterpriseAPIURL, env.ClusterID)
		req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			log.Error().Msgf("error creating http request %s", err)
			return fmt.Errorf("error creating http request %q: %w", endpoint, err)
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		counter := 0
		maxTries := 12
		output := bytes.Buffer{}
		for {
			res, err := httpClient.Do(req)
			if err != nil {
				if counter > maxTries {
					log.Error().Msgf("error in http call to API EE: url (%s) did not come up within 2 minutes %s", req.URL, err.Error())
					return fmt.Errorf("error in http call to API EE: url %q did not come up within 2 minutes: %w", req.URL, err)
				}
				counter++
				time.Sleep(10 * time.Second)
				continue
			}
			defer res.Body.Close()

			if res.StatusCode == http.StatusAccepted {
				// if we got a 201 but we can't read the page's body,
				// we still got a cluster, so we should ignore the error
				io.Copy(&output, res.Body)
				break
			}

			// if we get a non-201 status code, we need to retry unless we exceed the counter
			if counter > maxTries {
				log.Error().Msgf("unable to create default workload clusters and default environments %s: \n request: %s", res.Status, res.Request.URL)
				return fmt.Errorf("unable to create default workload clusters and default environments: API returned status %q", res.Status)
			}
		}

		log.Info().Msgf("cluster %q created: details: %s", cluster.ClusterName, output.String())
		time.Sleep(20 * time.Second)
	}
	return nil
}
