package rediscontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (p pipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("deleting resource")

	redisInstance := mg.(*exoscalev1.Redis)
	resp, err := p.exo.DeleteDbaasServiceWithResponse(ctx, redisInstance.GetInstanceName())
	if err != nil {
		return fmt.Errorf("cannot delete instance: %w", err)
	}
	log.V(1).Info("response", "body", string(resp.Body))
	return nil
}
