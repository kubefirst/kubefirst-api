package kubernetes

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateSecretsIfNotExist(t *testing.T) {
	tests := []struct {
		name            string
		existingSecrets []Secret
		inputSecrets    []Secret
	}{
		{
			name: "secret found and skipped",
			existingSecrets: []Secret{
				{Name: "test-secret", Namespace: "default", Contents: map[string]string{"token": "test-token"}},
			},
			inputSecrets: []Secret{
				{Name: "test-secret", Namespace: "default", Contents: map[string]string{"token": "test-token"}},
			},
		},
		{
			name:            "secret not found",
			existingSecrets: []Secret{},
			inputSecrets: []Secret{
				{Name: "test-secret", Namespace: "default", Contents: map[string]string{"token": "test-token"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fake clientset
			clientset := fake.NewSimpleClientset()

			// Create existing secrets
			for _, secret := range tt.existingSecrets {
				clientset.CoreV1().Secrets(secret.Namespace).Create(context.TODO(), createSecret(secret), metav1.CreateOptions{})
			}

			// Run the function
			err := CreateSecretsIfNotExist(context.TODO(), clientset, tt.inputSecrets)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}
		})
	}
}
