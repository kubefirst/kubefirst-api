package providerConfigs // nolint:revive // allowing temporarily for better code organization

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestK8sNamespaces(t *testing.T) {
	tests := []struct {
		name               string
		existingNamespaces []string
		wantErr            bool
	}{
		{
			name:               "No existing namespaces",
			existingNamespaces: []string{},
			wantErr:            false,
		},
		{
			name:               "Some existing namespaces",
			existingNamespaces: []string{"argocd", "argo"},
			wantErr:            false,
		},
		{
			name: "All namespaces exist",
			existingNamespaces: []string{
				"argocd",
				"argo",
				"atlantis",
				"chartmuseum",
				"cert-manager",
				"crossplane-system",
				"kubefirst",
				"external-dns",
				"external-secrets-operator",
				"vault",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewSimpleClientset()

			// Create existing namespaces
			for _, ns := range tt.existingNamespaces {
				clientset.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: ns},
				}, metav1.CreateOptions{})
			}

			// Run the function
			err := Namespaces(clientset)

			// Check for error
			if (err != nil) != tt.wantErr {
				t.Errorf("K8sNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
