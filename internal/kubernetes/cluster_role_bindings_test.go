package kubernetes

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateClusterRoleBindingsIfNotExist(t *testing.T) {
	tests := []struct {
		name                string
		clusterRoleBindings []ClusterRoleBinding
	}{
		{
			name: "Create new cluster role binding",
			clusterRoleBindings: []ClusterRoleBinding{
				{
					Name:     "test-role-binding",
					Subjects: []rbacv1.Subject{},
					RoleRef:  rbacv1.RoleRef{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			if err := CreateClusterRoleBindingsIfNotExist(context.Background(), k8s, tt.clusterRoleBindings); err != nil {
				t.Errorf("CreateClusterRoleBindingsIfNotExist() error = %v", err)
			}
		})
	}
}

func TestDeleteClusterRoleBinding(t *testing.T) {
	tests := []struct {
		name        string
		roleBinding string
	}{
		{
			name:        "Delete existing cluster role binding",
			roleBinding: "test-role-binding",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			// First create the role binding
			if err := CreateClusterRoleBindingsIfNotExist(context.Background(), k8s, []ClusterRoleBinding{{Name: tt.roleBinding, Subjects: []rbacv1.Subject{}, RoleRef: rbacv1.RoleRef{}}}); err != nil {
				t.Fatalf("Setup failed: CreateClusterRoleBindingsIfNotExist() error = %v", err)
			}

			// Then try to delete it
			if err := DeleteClusterRoleBinding(context.Background(), k8s, tt.roleBinding); err != nil {
				t.Errorf("DeleteClusterRoleBinding() error = %v", err)
			}
		})
	}
}
