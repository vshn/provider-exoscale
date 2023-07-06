package bucketcontroller

import (
	"context"
	"fmt"
	"net/http"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

var bucketExistsFn = func(ctx context.Context, mc *minio.Client, bucketName string) (bool, error) {
	return mc.BucketExists(ctx, bucketName)
}

// Observe implements managed.ExternalClient.
func (p *ProvisioningPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	bucket := fromManaged(mg)

	bucketName := bucket.GetBucketName()
	exists, err := bucketExistsFn(ctx, p.minioClient, bucketName)

	if err != nil {
		errResp := minio.ToErrorResponse(err)
		if errResp.StatusCode == http.StatusForbidden {
			return managed.ExternalObservation{}, errors.Wrap(err, "wrong credentials or bucket exists already, try changing bucket name")
		}
		if errResp.StatusCode == http.StatusMovedPermanently {
			return managed.ExternalObservation{}, errors.Wrap(err, "mismatching endpointURL and zone, or bucket exists already in a different region, try changing bucket name")
		}
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot determine whether bucket exists")
	}
	if _, hasAnnotation := bucket.Annotations[lockAnnotation]; hasAnnotation && exists {
		bucket.Status.AtProvider.BucketName = bucketName
		bucket.SetConditions(xpv1.Available())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
	} else if exists {
		return managed.ExternalObservation{}, fmt.Errorf("bucket exists already, try changing bucket name: %s", bucketName)
	}
	return managed.ExternalObservation{}, nil
}
