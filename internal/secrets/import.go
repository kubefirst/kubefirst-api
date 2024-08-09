/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/constants"
	"github.com/kubefirst/kubefirst-api/internal/env"
	"github.com/kubefirst/kubefirst-api/internal/k8s"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	"github.com/rs/zerolog/log"
)

func ImportClusterIfEmpty() (pkgtypes.Cluster, error) {
	// find the secret in mgmt cluster's kubefirst namespace and read import payload and clustername
	cluster := pkgtypes.Cluster{}
	env, _ := env.GetEnv(constants.SilenceGetEnv)

	if env.IsClusterZero == "true" {
		log.Info().Msg("IS_CLUSTER_ZERO is set to true, skipping import cluster logic.")
		return cluster, nil
	}

	kcfg := k8s.CreateKubeConfig(true, "")
	log.Info().Msg("reading secret kubefirst-initial-state to determine if import is needed")
	secData, err := k8s.ReadSecretV2Old(kcfg.Clientset, "kubefirst", "kubefirst-initial-state")
	if err != nil {
		log.Info().Msgf("error reading secret kubefirst-initial-state. %s", err)
		return cluster, err
	}

	jsonString, _ := MapToStructuredJSON(secData)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return cluster, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal([]byte(jsonData), &cluster)
	if err != nil {
		return cluster, fmt.Errorf("unable to cast cluster: %w", err)
	}

	log.Info().Msgf("import cluster secret discovered for cluster %s", cluster.ClusterName)

	// if you find a record bail
	_, err = GetCluster(kcfg.Clientset, cluster.ClusterName)
	if err != nil {
		log.Info().Stack().Msgf("did not find preexisting record for cluster %s. importing record.", cluster.ClusterName)

		// Create if entry does not exist
		err = InsertCluster(kcfg.Clientset, cluster)
		if err != nil {
			return cluster, fmt.Errorf("error inserting cluster %v: %s", cluster, err)
		}
		// log cluster
		log.Info().Msgf("inserted cluster record to db. adding default services. %s", cluster.ClusterName)

		return cluster, nil
	}

	log.Info().Msgf("cluster record for %s already exists - skipping", cluster.ClusterName)
	return pkgtypes.Cluster{}, nil
}
