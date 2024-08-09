package kubernetes

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateNamespacesIfNotExist(t *testing.T) {
	tests := []struct {
		name               string
		existingNamespaces []Namespace
		inputNamespaces    []Namespace
	}{
		{
			name: "namespace found and skipped",
			existingNamespaces: []Namespace{
				{Name: "test-namespace"},
			},
			inputNamespaces: []Namespace{
				{Name: "test-namespace"},
			},
		},
		{
			name:               "namespace not found",
			existingNamespaces: []Namespace{},
			inputNamespaces: []Namespace{
				{Name: "test-namespace"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewSimpleClientset()

			// Create existing namespaces
			for _, namespace := range tt.existingNamespaces {
				clientset.CoreV1().Namespaces().Create(context.TODO(), shortNamespaceToLong(namespace), metav1.CreateOptions{})
			}

			// Run the function
			err := CreateNamespacesIfNotExist(context.TODO(), clientset, tt.inputNamespaces)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
