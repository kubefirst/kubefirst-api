package kubernetes

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateServiceAccountsIfNotExist(t *testing.T) {
	tests := []struct {
		name                    string
		existingServiceAccounts []ServiceAccount
		inputServiceAccounts    []ServiceAccount
	}{
		{
			name: "service account found and skipped",
			existingServiceAccounts: []ServiceAccount{
				{Name: "test-service-account", Namespace: "default"},
			},
			inputServiceAccounts: []ServiceAccount{
				{Name: "test-service-account", Namespace: "default"},
			},
		},
		{
			name:                    "service account not found",
			existingServiceAccounts: []ServiceAccount{},
			inputServiceAccounts: []ServiceAccount{
				{Name: "test-service-account", Namespace: "default"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewSimpleClientset()

			// Create existing service accounts
			for _, serviceAccount := range tt.existingServiceAccounts {
				clientset.CoreV1().ServiceAccounts(serviceAccount.Namespace).Create(context.TODO(), createServiceAccount(serviceAccount), metav1.CreateOptions{})
			}

			// Run the function
			err := CreateServiceAccountsIfNotExist(context.TODO(), clientset, tt.inputServiceAccounts)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
