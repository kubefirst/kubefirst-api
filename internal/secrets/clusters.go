/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/k8s"
	pkgtypes "github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	secretName    = "kubefirst-clusters"
	clusterPrefix = "kubefirst-cluster"
)

// DeleteCluster
func DeleteCluster(clientSet *kubernetes.Clientset, clusterName string) error {
	err := DeleteSecretReference(clientSet, secretName, clusterName)
	if err != nil {
		return fmt.Errorf("error deleting cluster %s reference", clusterName)
	}

	err = k8s.DeleteSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, clusterName))
	if err != nil {
		return fmt.Errorf("error deleting cluster %s: %s", clusterName, err)
	}

	log.Info().Msgf("cluster deleted: %v", clusterName)

	return nil
}

// GetCluster
func GetCluster(clientSet *kubernetes.Clientset, clusterName string) (pkgtypes.Cluster, error) {
	cluster := pkgtypes.Cluster{}

	clusterSecret, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, clusterName))
	if err != nil {
		return cluster, fmt.Errorf("secret not found: %w", err)
	}
	jsonString, _ := MapToStructuredJSON(clusterSecret)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return cluster, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal([]byte(jsonData), &cluster)
	if err != nil {
		return cluster, fmt.Errorf("unable to cast cluster: %w", err)
	}

	return cluster, nil
}

// GetCluster
func GetClusters(clientSet *kubernetes.Clientset) ([]pkgtypes.Cluster, error) {
	clusterList := []pkgtypes.Cluster{}
	clusterReferenceList, _ := GetSecretReference(clientSet, secretName)
	for _, clusterName := range clusterReferenceList.List {
		cluster, _ := GetCluster(clientSet, clusterName)

		if cluster.ClusterName != "" {
			clusterList = append(clusterList, cluster)
		}
	}

	return clusterList, nil
}

// InsertCluster
func InsertCluster(clientSet *kubernetes.Clientset, cl pkgtypes.Cluster) error {
	_, err := GetSecretReference(clientSet, secretName)

	if err != nil {
		CreateSecretReference(clientSet, secretName, pkgtypes.SecretListReference{
			Name: "clusters",
			List: []string{cl.ClusterName},
		})
	} else {
		err = AddSecretReferenceItem(clientSet, secretName, cl.ClusterName)
		if err != nil {
			return err
		}
	}

	bytes, _ := json.Marshal(cl)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", clusterPrefix, cl.ClusterName),
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err = k8s.CreateSecretV2(clientSet, secretToCreate)
	if err != nil {
		return fmt.Errorf("error creating kubernetes secret: %w", err)
	}

	return nil
}

// UpdateCluster
func UpdateCluster(clientSet *kubernetes.Clientset, cluster pkgtypes.Cluster) error {
	bytes, _ := json.Marshal(cluster)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", clusterPrefix, cluster.ClusterName), secretValuesMap)
	if err != nil {
		return fmt.Errorf("error updating kubernetes secret: %w", err)
	}

	return nil
}
