package kubernetes

import (
	"context"
	"fmt"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Namespace struct {
	Name        string
	Annotations map[string]string
	Labels      map[string]string
}

func createNamespace(ctx context.Context, k8s kubernetes.Interface, namespaces []*v1.Namespace) error {
	for _, ns := range namespaces {
		_, err := k8s.CoreV1().Namespaces().Get(ctx, ns.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating namespace %s: %w", ns.Name, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving namespace %s: %w (%T)", ns.Name, err, err)
		}
	}
	return nil
}

func CreateNamespacesIfNotExist(ctx context.Context, k8s kubernetes.Interface, namespaces []Namespace) error {
	converted := make([]*v1.Namespace, 0, len(namespaces))

	for _, ns := range namespaces {
		converted = append(converted, shortNamespaceToLong(ns))
	}

	return createNamespace(ctx, k8s, converted)
}

func CreateNamespacesIfNotExistSimple(ctx context.Context, k8s kubernetes.Interface, namespaces []string) error {
	converted := make([]*v1.Namespace, 0, len(namespaces))

	for _, ns := range namespaces {
		converted = append(converted, &v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	}

	return createNamespace(ctx, k8s, converted)
}

func shortNamespaceToLong(opts Namespace) *v1.Namespace {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: opts.Name,
		},
	}

	if opts.Annotations != nil {
		namespace.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		namespace.Labels = opts.Labels
	}

	return namespace
}
