package iamkeycontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
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

	updateRole := exoscalesdk.UpdateIAMRoleRequest{
		Description: role.Description,
		Labels:      role.Labels,
		Permissions: role.Permissions,
	}
	_, err := p.exoscaleClient.UpdateIAMRole(ctx, iamKey.Status.AtProvider.RoleID, updateRole)

	return managed.ExternalUpdate{}, err
}
