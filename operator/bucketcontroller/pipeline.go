package bucketcontroller

import (
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ProvisioningPipeline provisions Buckets using S3 client.
type ProvisioningPipeline struct {
	recorder event.Recorder
	kube     client.Client

	minio *minio.Client
}

// NewProvisioningPipeline returns a new instance of ProvisioningPipeline.
func NewProvisioningPipeline(kube client.Client, recorder event.Recorder, minio *minio.Client) *ProvisioningPipeline {
	return &ProvisioningPipeline{
		kube:     kube,
		recorder: recorder,
		minio:    minio,
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.Bucket {
	return mg.(*exoscalev1.Bucket)
}
