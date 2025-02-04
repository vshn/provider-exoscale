package iamkeycontroller

import (
	"context"
	"errors"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	corev1 "k8s.io/api/core/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// KeyIDAnnotationKey is the annotation key where the IAMKey ID is stored.
	KeyIDAnnotationKey  = "exoscale.crossplane.io/key-id"
	RoleIDAnnotationKey = "exoscale.crossplane.io/role-id"
	// BucketResourceType is the resource type bucket to which the IAMKey has access to.
	BucketResourceType = "bucket"
	//SOSResourceDomain is the resource domain to which the IAMKey has access to.
	SOSResourceDomain = "sos"
)

// IAMKeyPipeline provisions IAMKeys on exoscale.com
type IAMKeyPipeline struct {
	kube           client.Client
	recorder       event.Recorder
	exoscaleClient *exoscalesdk.Client
	apiKey         string
	apiSecret      string
}

type pipelineContext struct {
	context.Context
	iamKey            *exoscalev1.IAMKey
	iamExoscaleKey    *exoscalesdk.AccessKey
	credentialsSecret *corev1.Secret
}

type IamRolesList struct {
	IamRoles []exoscalesdk.IAMRole `json:"iam-roles"`
}

type IamKeysList struct {
	IamKeys []exoscalesdk.IAMAPIKey `json:"api-keys"`
}

// NewPipeline returns a new instance of IAMKeyPipeline.
func NewPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client, apiKey, apiSecret string) *IAMKeyPipeline {
	return &IAMKeyPipeline{
		kube:           client,
		recorder:       recorder,
		exoscaleClient: exoscaleClient,
		apiKey:         apiKey,
		apiSecret:      apiSecret,
	}
}

func toConnectionDetails(iamKey *exoscalesdk.AccessKey) (managed.ConnectionDetails, error) {

	if iamKey.Key == "" {
		return nil, errors.New("iamKey key not found in connection details")
	}
	if iamKey.Secret == "" {
		return nil, errors.New("iamKey secret not found in connection details")
	}
	return map[string][]byte{
		exoscalev1.AccessKeyIDName:     []byte(iamKey.Key),
		exoscalev1.SecretAccessKeyName: []byte(iamKey.Secret),
	}, nil
}

func fromManaged(mg resource.Managed) *exoscalev1.IAMKey {
	return mg.(*exoscalev1.IAMKey)
}
