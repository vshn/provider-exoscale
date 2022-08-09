package iamkeycontroller

import (
	"context"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// KeyIDAnnotationKey is the annotation key where the IAMKey ID is stored.
	KeyIDAnnotationKey = "exoscale.crossplane.io/key-id"
	// BucketResourceType is the resource type bucket to which the IAMKey has access to.
	BucketResourceType = "bucket"
	//SOSResourceDomain is the resource domain to which the IAMKey has access to.
	SOSResourceDomain = "sos"
)

// IAMKeyPipeline provisions IAMKeys on exoscale.com
type IAMKeyPipeline struct {
	kube     client.Client
	recorder event.Recorder

	exoscaleClient    *exoscalesdk.Client
	exoscaleIAMKey    *exoscalesdk.IAMAccessKey
	credentialsSecret *corev1.Secret
}

// NewPipeline returns a new instance of IAMKeyPipeline.
func NewPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client) *IAMKeyPipeline {
	return &IAMKeyPipeline{
		kube:           client,
		recorder:       recorder,
		exoscaleClient: exoscaleClient,
	}
}

func hasSecretRef(iamKey *exoscalev1.IAMKey) pipeline.Predicate {
	return func(ctx context.Context) bool {
		return iamKey.Spec.WriteConnectionSecretToReference != nil
	}
}

func toConnectionDetails(iamKey *exoscalesdk.IAMAccessKey) managed.ConnectionDetails {
	return map[string][]byte{
		exoscalev1.AccessKeyIDName:     []byte(*iamKey.Key),
		exoscalev1.SecretAccessKeyName: []byte(*iamKey.Secret),
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.IAMKey {
	return mg.(*exoscalev1.IAMKey)
}
