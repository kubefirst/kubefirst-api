package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ServiceAccount struct {
	Name        string
	Namespace   string
	Automount   bool
	Annotations map[string]string
	Labels      map[string]string
}

func CreateServiceAccountsIfNotExist(ctx context.Context, k8s kubernetes.Interface, serviceAccounts []ServiceAccount) error {
	for _, sa := range serviceAccounts {
		serviceAccount := createServiceAccount(sa)

		_, err := k8s.CoreV1().ServiceAccounts(serviceAccount.Namespace).Get(ctx, serviceAccount.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.CoreV1().ServiceAccounts(serviceAccount.Namespace).Create(ctx, serviceAccount, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating service account %s in namespace %s: %w", serviceAccount.Name, serviceAccount.Namespace, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving service account %s in namespace %s: %w (%T)", serviceAccount.Name, serviceAccount.Namespace, err, err)
		}
	}
	return nil
}

func DeleteServiceAccount(ctx context.Context, k8s kubernetes.Interface, sa ServiceAccount) error {
	err := k8s.CoreV1().ServiceAccounts(sa.Namespace).Delete(ctx, sa.Name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting service account %s in namespace %s: %w", sa.Name, sa.Namespace, err)
	}
	return nil
}

func createServiceAccount(opts ServiceAccount) *v1.ServiceAccount {
	serviceAccount := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
	}

	if opts.Automount {
		serviceAccount.AutomountServiceAccountToken = &opts.Automount
	}

	if opts.Annotations != nil {
		serviceAccount.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		serviceAccount.Labels = opts.Labels
	}

	return serviceAccount
}
