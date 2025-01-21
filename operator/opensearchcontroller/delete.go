package opensearchcontroller

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *pipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("deleting resource")

	openSearch := mg.(*exoscalev1.OpenSearch)
	resp, err := p.exo.DeleteDBAASServiceOpensearch(ctx, openSearch.GetInstanceName())
	if err != nil {
		return fmt.Errorf("cannot delete OpenSearch: %w", err)
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return nil
}
