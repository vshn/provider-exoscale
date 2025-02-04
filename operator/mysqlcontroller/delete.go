package mysqlcontroller

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
	log.Info("deleting resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)
	resp, err := p.exo.DeleteDBAASServiceMysql(ctx, mySQLInstance.GetInstanceName())
	if err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("cannot delete mySQLInstance: %w", err)
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalDelete{}, nil
}
