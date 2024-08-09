package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Secret struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Contents    map[string]string
}

func CreateSecretsIfNotExist(ctx context.Context, k8s kubernetes.Interface, secrets []Secret) error {
	for _, s := range secrets {
		secret := createSecret(s)

		_, err := k8s.CoreV1().Secrets(secret.Namespace).Get(ctx, secret.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.CoreV1().Secrets(secret.Namespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating secret %s in namespace %s: %w", secret.Name, secret.Namespace, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving secret %s in namespace %s: %w (%T)", secret.Name, secret.Namespace, err, err)
		}
	}
	return nil
}

func createSecret(opts Secret) *v1.Secret {
	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		Data: map[string][]byte{},
	}

	if opts.Annotations != nil {
		secret.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		secret.Labels = opts.Labels
	}

	for key, value := range opts.Contents {
		secret.Data[key] = []byte(value)
	}

	return secret
}
