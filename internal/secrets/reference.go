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
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func GetSecretReference(clientSet *kubernetes.Clientset, secretName string) (pkgtypes.SecretListReference, error) {
	secretReference := pkgtypes.SecretListReference{}

	kubefirstSecrets, err := k8s.ReadSecretV2Old(clientSet, "kubefirst", secretName)
	if err != nil {
		return secretReference, fmt.Errorf("secret not found: %w", err)
	}
	jsonString, _ := MapToStructuredJSON(kubefirstSecrets)

	jsonData, err := json.Marshal(jsonString)
	if err != nil {
		return secretReference, fmt.Errorf("error marshalling json: %w", err)
	}

	err = json.Unmarshal([]byte(jsonData), &secretReference)
	if err != nil {
		return secretReference, fmt.Errorf("unable to cast secret reference: %w", err)
	}

	return secretReference, nil
}

func DeleteSecretReference(clientSet *kubernetes.Clientset, secretName string, valueToDelete string) error {
	filteredReferenceList := pkgtypes.SecretListReference{}
	ReferenceList, _ := GetSecretReference(clientSet, secretName)
	filteredReferenceList.Name = ReferenceList.Name

	for _, referenceClusterName := range ReferenceList.List {
		if referenceClusterName != valueToDelete {
			filteredReferenceList.List = append(filteredReferenceList.List, referenceClusterName)
		}
	}

	err := UpdateSecretReference(clientSet, secretName, filteredReferenceList)
	if err != nil {
		return err
	}

	return nil
}

// UpdateSecretReference
func UpdateSecretReference(clientSet *kubernetes.Clientset, secretName string, secretReference pkgtypes.SecretListReference) error {
	bytes, _ := json.Marshal(secretReference)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	err := k8s.UpdateSecretV2(clientSet, "kubefirst", secretName, secretValuesMap)
	if err != nil {
		return fmt.Errorf("error updating secret reference: %w", err)
	}

	return nil
}

func CreateSecretReference(clientSet *kubernetes.Clientset, secretName string, secretReference pkgtypes.SecretListReference) error {
	bytes, _ := json.Marshal(secretReference)
	secretValuesMap, _ := ParseJSONToMap(string(bytes))

	secretToCreate := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: "kubefirst",
		},
		Data: secretValuesMap,
	}

	err := k8s.CreateSecretV2(clientSet, secretToCreate)
	if err != nil {
		return fmt.Errorf("error creating secret reference: %w", err)
	}

	return nil
}

func AddSecretReferenceItem(clientSet *kubernetes.Clientset, secretName string, valueToAdd string) error {
	secretReference, _ := GetSecretReference(clientSet, secretName)
	secretReference.List = append(secretReference.List, valueToAdd)

	err := UpdateSecretReference(clientSet, secretName, secretReference)
	if err != nil {
		return err
	}

	return nil
}
