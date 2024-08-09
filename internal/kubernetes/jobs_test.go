package kubernetes

import (
	"context"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateJobsIfNotExist(t *testing.T) {
	tests := []struct {
		name string
		jobs []Job
	}{
		{
			name: "Create new job",
			jobs: []Job{
				{
					Name:      "test-job",
					Namespace: "default",
					Spec:      batchv1.JobSpec{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			if err := CreateJobsIfNotExist(context.Background(), k8s, tt.jobs); err != nil {
				t.Errorf("CreateJobsIfNotExist() error = %v", err)
			}
		})
	}
}

func TestRecreateJobs(t *testing.T) {
	tests := []struct {
		name string
		jobs []Job
	}{
		{
			name: "Recreate existing job",
			jobs: []Job{
				{
					Name:      "test-job",
					Namespace: "default",
					Spec:      batchv1.JobSpec{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k8s := fake.NewSimpleClientset()

			// First create the job
			if err := CreateJobsIfNotExist(context.Background(), k8s, tt.jobs); err != nil {
				t.Errorf("CreateJobsIfNotExist() error = %v", err)
			}

			// Then try to recreate it
			if err := RecreateJobs(context.Background(), k8s, tt.jobs); err != nil {
				t.Errorf("RecreateJobs() error = %v", err)
			}
		})
	}
}
