package postgresqlcontroller

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *pipeline) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	pgInstance := mg.(*exoscalev1.PostgreSQL)
	resp, err := p.exo.DeleteDBAASServicePG(ctx, pgInstance.GetInstanceName())
	if err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("cannot delete instance: %w", err)
	}
	log.V(1).Info("Response when deleting", "message", resp.Message)
	return managed.ExternalDelete{}, nil
}
