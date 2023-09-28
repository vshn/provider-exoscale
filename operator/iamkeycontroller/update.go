package iamkeycontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exov1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
// exoscale.com does not allow any updates on IAM keys.
func (p *IAMKeyPipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Updating role")

	iamKey := fromManaged(mg)
	iamKey.SetConditions(exov1.Updating())

	role := createRole(iamKey.Spec.ForProvider.KeyName, iamKey.Spec.ForProvider.Services.SOS.Buckets)

	_, err := executeRequest(ctx, "PUT", iamKey.Spec.ForProvider.Zone, "/v2/iam-role/"+iamKey.Status.AtProvider.RoleID+":policy", p.apiKey, p.apiSecret, role.Policy)

	return managed.ExternalUpdate{}, err
}
