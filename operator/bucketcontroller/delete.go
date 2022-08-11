package bucketcontroller

import (
	"context"
	"fmt"
	"github.com/vshn/provider-exoscale/operator/steps"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *ProvisioningPipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	bucket := fromManaged(mg)
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.If(hasDeleteAllPolicy(*bucket),
				pipeline.NewStepFromFunc("delete all objects", p.deleteAllObjectsFn(*bucket))),
			pipeline.NewStepFromFunc("delete bucket", p.deleteS3BucketFn(*bucket)),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(*bucket)),
		).RunWithContext(ctx)

	return errors.Wrap(result.Err(), "cannot deprovision bucket")
}

func hasDeleteAllPolicy(bucket exoscalev1.Bucket) pipeline.Predicate {
	return func(ctx context.Context) bool {
		return bucket.Spec.ForProvider.BucketDeletionPolicy == exoscalev1.DeleteAll
	}
}

func (p *ProvisioningPipeline) deleteAllObjectsFn(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log := controllerruntime.LoggerFrom(ctx)
		bucketName := bucket.Status.AtProvider.BucketName

		objectsCh := make(chan minio.ObjectInfo)

		// Send object names that are needed to be removed to objectsCh
		go func() {
			defer close(objectsCh)
			for object := range p.minio.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
				if object.Err != nil {
					log.V(1).Info("warning: cannot list object", "key", object.Key, "error", object.Err)
					continue
				}
				objectsCh <- object
			}
		}()

		for obj := range p.minio.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{GovernanceBypass: true}) {
			return fmt.Errorf("object %q cannot be removed: %w", obj.ObjectName, obj.Err)
		}
		return nil
	}
}

// deleteS3Bucket deletes the bucket.
// NOTE: The removal fails if there are still objects in the bucket.
// This func does not recursively delete all objects beforehand.
func (p *ProvisioningPipeline) deleteS3BucketFn(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		bucketName := bucket.Status.AtProvider.BucketName
		err := p.minio.RemoveBucket(ctx, bucketName)
		return err
	}
}

func (p *ProvisioningPipeline) emitDeletionEventFn(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		p.recorder.Event(&bucket, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Deleted",
			Message: "Bucket deleted",
		})
		return nil
	}
}
