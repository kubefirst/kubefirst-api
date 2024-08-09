package controller

import (
	"context"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRestartDeployment(t *testing.T) {
	client := fake.NewSimpleClientset()

	namespace := "default"

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-deployment",
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{},
				},
			},
		},
	}

	ctx := context.Background()

	// kubectl apply -f deployment.yaml
	_, err := client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error creating deployment: %v", err)
	}

	// kubectl rollout restart deployment example-deployment
	if err := RestartDeployment(ctx, client, namespace, deployment.Name); err != nil {
		t.Fatalf("error restarting deployment: %v", err)
	}

	// kubectl get deployment ${name} -o yaml
	deployment, err = client.AppsV1().Deployments(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting deployment: %v", err)
	}

	got, found := deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"]
	if !found {
		t.Fatalf("expected annotation restartedAt to be set, got nothing")
	}

	if got == "" {
		t.Fatalf("expected annotation restartedAt to be set, got empty string")
	}

	t1, err := time.Parse(time.RFC3339, got)
	if err != nil {
		t.Fatalf("error parsing time %q: %v", got, err)
	}

	t2 := time.Now()

	if t2.Sub(t1) > 1*time.Second {
		t.Fatalf("expected annotation restartedAt to be set within 1 second, got %v", t2.Sub(t1))
	}
}
