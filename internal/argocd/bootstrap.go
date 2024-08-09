/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package argocd

import (
	"context"
	"fmt"

	"github.com/kubefirst/kubefirst-api/internal/k8s"
	kube "github.com/kubefirst/kubefirst-api/internal/kubernetes"
	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// ApplyArgoCDKustomize
func ApplyArgoCDKustomize(clientset *kubernetes.Clientset, argoCDInstallPath string) error {
	var (
		enabled   = true
		name      = "argocd-bootstrap"
		namespace = "argocd"
		ctx       = context.Background()
	)

	// Create Namespace
	if err := kube.CreateNamespacesIfNotExistSimple(ctx, clientset, []string{namespace}); err != nil {
		log.Error().Msgf("error creating namespace: %s", err)
		return fmt.Errorf("error creating namespace: %w", err)
	}

	// Create ServiceAccount
	serviceAccounts := []kube.ServiceAccount{{
		Name:      name,
		Namespace: namespace,
		Automount: enabled,
	}}

	if err := kube.CreateServiceAccountsIfNotExist(ctx, clientset, serviceAccounts); err != nil {
		log.Error().Msgf("error creating service account: %s", err)
		return fmt.Errorf("error creating service account: %w", err)
	}

	// Create ClusterRole
	clusterRole := kube.ClusterRole{
		Name: name,
		Rules: []rbacv1.PolicyRule{{
			Verbs:     []string{"*"},
			APIGroups: []string{"*"},
			Resources: []string{"*"},
		}},
	}
	if err := kube.CreateClusterRolesIfNotExist(ctx, clientset, []kube.ClusterRole{clusterRole}); err != nil {
		log.Error().Msgf("error creating cluster role: %s", err)
		return fmt.Errorf("error creating cluster role: %w", err)
	}

	crb := kube.ClusterRoleBinding{
		Name: name,
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      name,
			Namespace: namespace,
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     name,
		},
	}
	if err := kube.CreateClusterRoleBindingsIfNotExist(ctx, clientset, []kube.ClusterRoleBinding{crb}); err != nil {
		log.Error().Msgf("error creating cluster role binding: %s", err)
		return fmt.Errorf("error creating cluster role binding: %w", err)
	}

	// Create Job
	backoffLimit := int32(1)
	job := kube.Job{
		Name:      "kustomize-apply-argocd",
		Namespace: namespace,
		Spec: batchv1.JobSpec{
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{},
				Spec: v1.PodSpec{
					Containers: []v1.Container{{
						Name:  "main",
						Image: "bitnami/kubectl:1.28",
						Command: []string{
							"/bin/sh",
							"-c",
							fmt.Sprintf("kubectl apply -k '%s'", argoCDInstallPath),
						},
					}},
					ServiceAccountName: name,
					RestartPolicy:      "Never",
				},
			},
			BackoffLimit: &backoffLimit,
		},
	}
	if err := kube.RecreateJobs(ctx, clientset, []kube.Job{job}); err != nil {
		log.Error().Msgf("error creating job: %s", err)
		return fmt.Errorf("error creating job: %w", err)
	}

	log.Info().Msg("created argocd bootstrap job")

	// Wait for the Job to finish
	_, err := k8s.WaitForJobComplete(clientset, job.Name, job.Namespace, 240)
	if err != nil {
		log.Error().Msgf("could not run argocd bootstrap job: %s", err)
		return fmt.Errorf("could not run argocd bootstrap job: %w", err)
	}

	// Cleanup
	if err := kube.DeleteServiceAccount(ctx, clientset, kube.ServiceAccount{Name: name, Namespace: namespace}); err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap service account %s - manual removal is required", name)
		return fmt.Errorf("could not clean up argocd bootstrap service account %s - manual removal is required: %w", name, err)
	}

	if err := kube.DeleteClusterRole(ctx, clientset, clusterRole.Name); err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role %s - manual removal is required", name)
		return fmt.Errorf("could not clean up argocd bootstrap cluster role %s - manual removal is required: %w", name, err)
	}

	if err := kube.DeleteClusterRoleBinding(ctx, clientset, crb.Name); err != nil {
		log.Error().Msgf("could not clean up argocd bootstrap cluster role binding %s - manual removal is required", name)
		return fmt.Errorf("could not clean up argocd bootstrap cluster role binding %s - manual removal is required: %w", name, err)
	}

	return nil
}
