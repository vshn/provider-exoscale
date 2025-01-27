package bucketcontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *ProvisioningPipeline) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	bucket := fromManaged(mg)
	pctx := &pipelineContext{Context: ctx, bucket: bucket}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.When(hasDeleteAllPolicy,
				"delete all objects", p.deleteAllObjects,
			),
			pipe.NewStep("delete bucket", p.deleteS3Bucket),
			pipe.NewStep("emit event", p.emitDeletionEvent),
		)
	err := pipe.RunWithContext(pctx)
	return managed.ExternalDelete{}, errors.Wrap(err, "cannot deprovision bucket")
}

func hasDeleteAllPolicy(ctx *pipelineContext) bool {
	return ctx.bucket.Spec.ForProvider.BucketDeletionPolicy == exoscalev1.DeleteAll
}

func (p *ProvisioningPipeline) deleteAllObjects(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	bucketName := ctx.bucket.Status.AtProvider.BucketName

	objectsCh := make(chan minio.ObjectInfo)

	// Send object names that are needed to be removed to objectsCh
	go func() {
		defer close(objectsCh)
		for object := range p.minioClient.ListObjects(ctx, bucketName, minio.ListObjectsOptions{Recursive: true}) {
			if object.Err != nil {
				log.V(1).Info("warning: cannot list object", "key", object.Key, "error", object.Err)
				continue
			}
			objectsCh <- object
		}
	}()

	bypassGovernance, err := p.isBucketLockEnabled(ctx, bucketName)
	if err != nil {
		log.Error(err, "not able to determine ObjectLock status for bucket", "bucket", bucketName)
	}

	for obj := range p.minioClient.RemoveObjects(ctx, bucketName, objectsCh, minio.RemoveObjectsOptions{GovernanceBypass: bypassGovernance}) {
		return fmt.Errorf("object %q cannot be removed: %w", obj.ObjectName, obj.Err)
	}
	return nil
}

func (p *ProvisioningPipeline) isBucketLockEnabled(ctx context.Context, bucketName string) (bool, error) {
	_, mode, _, _, err := p.minioClient.GetObjectLockConfig(ctx, bucketName)
	if err != nil && err.Error() == "Object Lock configuration does not exist for this bucket" {
		return false, nil
	} else if err != nil {
		return false, err
	}
	// If there's an objectLockConfig it could still be disabled...
	return mode != nil, nil
}

// deleteS3Bucket deletes the bucket.
// NOTE: The removal fails if there are still objects in the bucket.
// This func does not recursively delete all objects beforehand.
func (p *ProvisioningPipeline) deleteS3Bucket(ctx *pipelineContext) error {
	bucketName := ctx.bucket.Status.AtProvider.BucketName
	err := p.minioClient.RemoveBucket(ctx, bucketName)
	return err
}

func (p *ProvisioningPipeline) emitDeletionEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.bucket, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Deleted",
		Message: "Bucket deleted",
	})
	return nil
}
