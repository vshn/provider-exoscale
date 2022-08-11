package bucketcontroller

import (
	"context"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/steps"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *ProvisioningPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	bucket := fromManaged(mg)
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("create bucket", p.createS3Bucket(*bucket)),
			pipeline.NewStepFromFunc("emit event", p.emitCreationEvent(*bucket)),
		).RunWithContext(ctx)

	return managed.ExternalCreation{}, errors.Wrap(result.Err(), "cannot create Bucket")
}

// createS3Bucket creates a new bucket and sets the name in the status.
// If the bucket already exists, and we have permissions to access it, no error is returned and the name is set in the status.
// If the bucket exists, but we don't own it, an error is returned.
func (p *ProvisioningPipeline) createS3Bucket(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		bucketName := bucket.GetBucketName()
		err := p.minio.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: bucket.Spec.ForProvider.Zone})

		if err != nil {
			// Check to see if we already own this bucket (which happens if we run this twice)
			exists, errBucketExists := p.minio.BucketExists(ctx, bucketName)
			if errBucketExists == nil && exists {
				return nil
			} else {
				// someone else might have created the bucket
				return err
			}
		}
		return nil
	}
}

func (p *ProvisioningPipeline) emitCreationEvent(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		p.recorder.Event(&bucket, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Created",
			Message: "Bucket successfully created",
		})
		return nil
	}
}
