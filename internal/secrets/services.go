/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package secrets

import (
	"encoding/json"
	"fmt"

	"github.com/konstructio/kubefirst-api/internal/k8s"
	"github.com/konstructio/kubefirst-api/pkg/types"
	log "github.com/rs/zerolog/log"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const kubefirstServicesPrefix = "kubefirst-service"

// CreateClusterServiceList adds an entry for a cluster to the service list
func CreateClusterServiceList(clientSet kubernetes.Interface, clusterName string) error {
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("error creating kubernetes service secret: %w", err)
	}

	if clusterServices != nil {
		log.Info().Msgf("cluster service list record for %s already exists - skipping", clusterName)
		return nil
	}

	clusterServices = &types.ClusterServiceList{
		ClusterName: clusterName,
		Services:    []types.Service{},
	}

	bytes, err := json.Marshal(clusterServices)
	if err != nil {
		return fmt.Errorf("error marshalling json: %w", err)
	}

	secretValuesMap, err := ParseJSONToMap(string(bytes))
	if err != nil {
		return fmt.Errorf("error parsing json: %w", err)
	}

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName),
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	if err := k8s.CreateSecretV2(clientSet, secretToCreate); err != nil {
		return fmt.Errorf("error creating kubernetes service secret: %w", err)
	}

	return nil
}

// DeleteClusterServiceListEntry removes a service entry from a cluster's service list
func DeleteClusterServiceListEntry(clientSet kubernetes.Interface, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("error deleting service list entry %q: secret not found: %w", def.Name, err)
		}

		return fmt.Errorf("error deleting service list entry %s: %w", def.Name, err)
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
		return fmt.Errorf("error deleting service list entry %s: %w", def.Name, err)
	}

	log.Info().Msgf("service deleted: %v", def.Name)
	return nil
}

// GetService returns a single service associated with a given cluster
func GetService(clientSet kubernetes.Interface, clusterName string, serviceName string) (types.Service, error) {
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
func GetServices(clientSet kubernetes.Interface, clusterName string) (*types.ClusterServiceList, error) {
	kubefirstSecrets, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", fmt.Sprintf("%s-%s", kubefirstServicesPrefix, clusterName))
	if err != nil {
		return nil, fmt.Errorf("error reading kubernetes service secret %s: %w", clusterName, err)
	}

	jsonString, err := MapToStructuredJSON(kubefirstSecrets)
	if err != nil {
		return nil, fmt.Errorf("error parsing json: %w", err)
	}

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %w", err)
	}

	clusterServices := types.ClusterServiceList{}
	err = json.Unmarshal(jsonData, &clusterServices)
	if err != nil {
		return nil, fmt.Errorf("unable to cast environment: %w", err)
	}

	return &clusterServices, nil
}

// InsertClusterServiceListEntry appends a service entry for a cluster's service list
func InsertClusterServiceListEntry(clientSet kubernetes.Interface, clusterName string, def *types.Service) error {
	// Find
	clusterServices, err := GetServices(clientSet, clusterName)
	if err != nil {
		return fmt.Errorf("error adding service list entry %s: %w", def.Name, err)
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
		return fmt.Errorf("error adding service list entry %s: %w", def.Name, err)
	}

	log.Info().Msgf("service added: %v", def.Name)
	return nil
}
