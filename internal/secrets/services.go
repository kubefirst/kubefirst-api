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
	"github.com/kubefirst/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const kubefirstServicesPrefix = "kubefirst-service"

// CreateClusterServiceList adds an entry for a cluster to the service list
func CreateClusterServiceList(clientSet *kubernetes.Clientset, clusterName string) error {
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil {
		return fmt.Errorf("error creating kubernetes service secret: %w", err)
	}

	if clusterServices.ClusterName == "" {
		clusterServices = types.ClusterServiceList{
			ClusterName: clusterName,
			Services:    []types.Service{},
		}

		bytes, _ := json.Marshal(clusterServices)
		secretValuesMap, _ := ParseJSONToMap(string(bytes))

		secretToCreate := &v1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName),
				Namespace: "kubefirst",
			},
			Data: secretValuesMap,
		}

		err := k8s.CreateSecretV2(clientSet, secretToCreate)
		if err != nil {
			return fmt.Errorf("error creating kubernetes service secret: %w", err)
		}
	}

	log.Info().Msgf("cluster service list record for %s already exists - skipping", clusterName)
	return nil
}

// DeleteClusterServiceListEntry removes a service entry from a cluster's service list
func DeleteClusterServiceListEntry(clientSet *kubernetes.Clientset, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil {
		return fmt.Errorf("error deleting service list entry %s: %s", def.Name, err)
	}

	filteredServiceList := []types.Service{}

	for _, service := range clusterServices.Services {
		if service.Name != def.Name {
			filteredServiceList = append(filteredServiceList, service)
		}
	}

	clusterServices.Services = filteredServiceList

	bytes, err := json.Marshal(clusterServices)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json: %w", err)
	}

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName), secretValuesMap)
	if err != nil {
		return fmt.Errorf("error deleting service list entry %s: %s", def.Name, err)
	}

	log.Info().Msgf("service deleted: %v", def.Name)
	return nil
}

// GetService returns a single service associated with a given cluster
func GetService(clientSet *kubernetes.Clientset, clusterName string, serviceName string) (types.Service, error) {
	// Find
	clusterServices, _ := GetServices(clientSet, clusterName)

	for _, service := range clusterServices.Services {
		if service.Name == serviceName {
			return service, nil
		}
	}

	return types.Service{}, fmt.Errorf("could not find service %s for cluster %s", serviceName, clusterName)
}

// GetServices returns services associated with a given cluster
func GetServices(clientSet *kubernetes.Clientset, clusterName string) (types.ClusterServiceList, error) {
	clusterServices := types.ClusterServiceList{}

	kubefirstSecrets, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName))
	if err != nil {
		return clusterServices, fmt.Errorf("error reading kubernetes service secret %s: %w", clusterName, err)
	}

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)
	if err != nil {
		return clusterServices, fmt.Errorf("error parsing json: %w", err)
	}

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return clusterServices, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal([]byte(jsonData), &clusterServices)
	if err != nil {
		return clusterServices, fmt.Errorf("unable to cast environment: %w", err)
	}

	return clusterServices, nil
}

// InsertClusterServiceListEntry appends a service entry for a cluster's service list
func InsertClusterServiceListEntry(clientSet *kubernetes.Clientset, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil {
		return fmt.Errorf("error adding service list entry %s: %s", def.Name, err)
	}

	clusterServices.Services = append(clusterServices.Services, *def)

	bytes, err := json.Marshal(clusterServices)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json: %w", err)
	}

	err = k8s.UpdateSecretV2(clientSet, "kubefirst", fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName), secretValuesMap)
	if err != nil {
		return fmt.Errorf("error adding service list entry %s: %s", def.Name, err)
	}

	log.Info().Msgf("service added: %v", def.Name)
	return nil
}
