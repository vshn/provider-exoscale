package bucketcontroller

import (
	"context"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *ProvisioningPipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	bucket := fromManaged(mg)

	bucketName := bucket.GetBucketName()
	exists, err := p.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot determine whether bucket exists")
	}
	if exists {
		bucket.Status.AtProvider.BucketName = bucketName
		bucket.SetConditions(xpv1.Available())
		return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true, ConnectionDetails: toConnectionDetails(bucket)}, nil
	}
	return managed.ExternalObservation{}, nil
}

func toConnectionDetails(bucket *exoscalev1.Bucket) managed.ConnectionDetails {
	return map[string][]byte{
		EndpointName: []byte(bucket.Spec.ForProvider.EndpointURL),
		ZoneName:     []byte(bucket.Spec.ForProvider.Zone),
		BucketName:   []byte(bucket.Spec.ForProvider.BucketName),
	}
}
