package bucketcontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *ProvisioningPipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Creating resource")

	bucket := fromManaged(mg)
	pctx := &pipelineContext{Context: ctx, bucket: bucket}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("create bucket", p.createS3Bucket),
			pipe.NewStep("set lock", p.setLock),
			pipe.NewStep("emit event", p.emitCreationEvent),
		)
	err := pipe.RunWithContext(pctx)

	return managed.ExternalCreation{}, errors.Wrap(err, "cannot provision bucket")
}

// createS3Bucket creates a new bucket and sets the name in the status.
// If the bucket already exists, and we have permissions to access it, no error is returned and the name is set in the status.
// If the bucket exists, but we don't own it, an error is returned.
func (p *ProvisioningPipeline) createS3Bucket(ctx *pipelineContext) error {
	bucketName := ctx.bucket.GetBucketName()
	err := p.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: ctx.bucket.Spec.ForProvider.Zone})

	if err != nil {
		// Check to see if we already own this bucket (which happens if we run this twice)
		exists, errBucketExists := p.minioClient.BucketExists(ctx, bucketName)
		if errBucketExists == nil && exists {
			return nil
		} else {
			// someone else might have created the bucket
			return err
		}
	}
	return nil
}

// setLock sets an annotation that tells the Observe func that we have successfully created the bucket.
// Without it, another resource that has the same bucket name might "adopt" the same bucket, causing 2 resources managing 1 bucket.
func (p *ProvisioningPipeline) setLock(ctx *pipelineContext) error {
	if ctx.bucket.Annotations == nil {
		ctx.bucket.Annotations = map[string]string{}
	}
	ctx.bucket.Annotations[lockAnnotation] = "claimed"
	return nil
}

func (p *ProvisioningPipeline) emitCreationEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.bucket, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Created",
		Message: "Bucket successfully created",
	})
	return nil
}
