package mysqlcontroller

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

	mySQLInstance := mg.(*exoscalev1.MySQL)
	resp, err := p.exo.DeleteDbaasServiceWithResponse(ctx, mySQLInstance.GetInstanceName())
	if err != nil {
		return fmt.Errorf("cannot delete mySQLInstance: %w", err)
	}
	log.V(1).Info("response", "json", string(resp.Body))
	return nil
}
