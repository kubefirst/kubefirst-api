package kubernetes

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Job struct {
	Name        string
	Namespace   string
	Annotations map[string]string
	Labels      map[string]string
	Spec        batchv1.JobSpec
}

func CreateJobsIfNotExist(ctx context.Context, k8s kubernetes.Interface, jobs []Job) error {
	for _, j := range jobs {
		job := createJob(j)

		_, err := k8s.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err := k8s.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
					return fmt.Errorf("error creating Job %s: %w", job.Name, err)
				}

				return nil
			}

			return fmt.Errorf("error retrieving Job %s: %w (%T)", job.Name, err, err)
		}
	}
	return nil
}

func RecreateJobs(ctx context.Context, k8s kubernetes.Interface, jobs []Job) error {
	for _, j := range jobs {
		job := createJob(j)

		_, err := k8s.BatchV1().Jobs(job.Namespace).Get(ctx, job.Name, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("error retrieving Job %s: %w (%T)", job.Name, err, err)
			}
		} else {
			if err := k8s.BatchV1().Jobs(job.Namespace).Delete(ctx, job.Name, metav1.DeleteOptions{}); err != nil {
				return fmt.Errorf("error deleting Job %s: %w", job.Name, err)
			}
		}

		if _, err := k8s.BatchV1().Jobs(job.Namespace).Create(ctx, job, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("error creating Job %s: %w", job.Name, err)
		}
	}
	return nil
}

func createJob(opts Job) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      opts.Name,
			Namespace: opts.Namespace,
		},
		Spec: opts.Spec,
	}

	if opts.Annotations != nil {
		job.Annotations = opts.Annotations
	}

	if opts.Labels != nil {
		job.Labels = opts.Labels
	}

	return job
}
