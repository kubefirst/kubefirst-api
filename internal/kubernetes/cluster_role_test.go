package kubernetes

import (
	"context"
	"testing"

	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateClusterRolesIfNotExist(t *testing.T) {
	tests := []struct {
		name  string
		roles []ClusterRole
	}{
		{
			name: "Create new cluster role",
			roles: []ClusterRole{
				{
					Name:  "test-role",
					Rules: []rbacv1.PolicyRule{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			if err := CreateClusterRolesIfNotExist(context.Background(), k8s, tt.roles); err != nil {
				t.Errorf("CreateClusterRolesIfNotExist() error = %v", err)
			}
		})
	}
}

func TestDeleteClusterRole(t *testing.T) {
	tests := []struct {
		name string
		role string
	}{
		{
			name: "Delete existing cluster role",
			role: "test-role",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			// First create the role
			if err := CreateClusterRolesIfNotExist(context.Background(), k8s, []ClusterRole{{Name: tt.role, Rules: []rbacv1.PolicyRule{}}}); err != nil {
				t.Fatalf("Setup failed: CreateClusterRolesIfNotExist() error = %v", err)
			}

			// Then try to delete it
			if err := DeleteClusterRole(context.Background(), k8s, tt.role); err != nil {
				t.Errorf("DeleteClusterRole() error = %v", err)
			}
		})
	}
}
