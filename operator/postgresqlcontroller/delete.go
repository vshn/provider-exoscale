package postgresqlcontroller

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *Pipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	pgInstance := fromManaged(mg)
	resp, err := p.exo.DeleteDbaasServiceWithResponse(ctx, pgInstance.GetInstanceName())
	if err != nil {
		return fmt.Errorf("cannot delete instance: %w", err)
	}
	log.V(1).Info("Response when deleting", "json", resp.JSON200)
	return nil
}
