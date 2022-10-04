package postgresqlcontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *Pipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	//pgInstance := fromManaged(mg)
	return nil
}
