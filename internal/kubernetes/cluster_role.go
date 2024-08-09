package kubernetes

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClusterRole struct {
	Name        string
	Annotations map[string]string
	Labels      map[string]string
	Rules       []rbacv1.PolicyRule
}

func CreateClusterRolesIfNotExist(ctx context.Context, k8s kubernetes.Interface, clusterRoles []ClusterRole) error {
	for _, c := range clusterRoles {
		clusterRole := createClusterRole(c)

		_, err := k8s.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.RbacV1().ClusterRoles().Create(ctx, clusterRole, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating ClusterRole %s: %w", clusterRole.Name, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving ClusterRole %s: %w (%T)", clusterRole.Name, err, err)
		}
	}
	return nil
}

func DeleteClusterRole(ctx context.Context, k8s kubernetes.Interface, clusterRoleName string) error {
	err := k8s.RbacV1().ClusterRoles().Delete(ctx, clusterRoleName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting ClusterRole %s: %w", clusterRoleName, err)
	}
	return nil
}

func createClusterRole(opts ClusterRole) *rbacv1.ClusterRole {
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.Name,
		},
		Rules: opts.Rules,
	}

	if opts.Annotations != nil {
		clusterRole.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		clusterRole.Labels = opts.Labels
	}

	return clusterRole
}
