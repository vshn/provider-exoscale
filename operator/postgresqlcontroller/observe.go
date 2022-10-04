package postgresqlcontroller

import (
	"context"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *Pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	//pgInstance := fromManaged(mg)
	return managed.ExternalObservation{}, nil
}
