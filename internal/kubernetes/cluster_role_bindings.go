package kubernetes

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type ClusterRoleBinding struct {
	Name        string
	Annotations map[string]string
	Labels      map[string]string
	Subjects    []rbacv1.Subject
	RoleRef     rbacv1.RoleRef
}

func CreateClusterRoleBindingsIfNotExist(ctx context.Context, k8s kubernetes.Interface, clusterRoleBindings []ClusterRoleBinding) error {
	for _, c := range clusterRoleBindings {
		clusterRoleBinding := createClusterRoleBinding(c)

		_, err := k8s.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.RbacV1().ClusterRoleBindings().Create(ctx, clusterRoleBinding, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating ClusterRoleBinding %s: %w", clusterRoleBinding.Name, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving ClusterRoleBinding %s: %w (%T)", clusterRoleBinding.Name, err, err)
		}
	}
	return nil
}

func DeleteClusterRoleBinding(ctx context.Context, k8s kubernetes.Interface, clusterRoleBindingName string) error {
	err := k8s.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("error deleting ClusterRoleBinding %s: %w", clusterRoleBindingName, err)
	}
	return nil
}

func createClusterRoleBinding(opts ClusterRoleBinding) *rbacv1.ClusterRoleBinding {
	clusterRoleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.Name,
		},
		Subjects: opts.Subjects,
		RoleRef:  opts.RoleRef,
	}

	if opts.Annotations != nil {
		clusterRoleBinding.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		clusterRoleBinding.Labels = opts.Labels
	}

	return clusterRoleBinding
}
