package bucketcontroller

import (
	"context"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// EndpointName is the endpoint field name in credential secret
	EndpointName = "ENDPOINT"
	// ZoneName is the region field name in credential secret
	ZoneName = "ZONE"
	// BucketName is the bucket field name in credential secret
	BucketName = "BUCKET_NAME"
)

// ProvisioningPipeline provisions Buckets using S3 client.
type ProvisioningPipeline struct {
	recorder    event.Recorder
	kube        client.Client
	minioClient *minio.Client
}

type pipelineContext struct {
	context.Context
	bucket                 *exoscalev1.Bucket
	IAMKeyCredentialSecret *corev1.Secret
}

// NewProvisioningPipeline returns a new instance of ProvisioningPipeline.
func NewProvisioningPipeline(kube client.Client, recorder event.Recorder, minioClient *minio.Client) *ProvisioningPipeline {
	return &ProvisioningPipeline{
		kube:        kube,
		recorder:    recorder,
		minioClient: minioClient,
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.Bucket {
	return mg.(*exoscalev1.Bucket)
}
