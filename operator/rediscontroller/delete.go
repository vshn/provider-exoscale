package rediscontroller

import (
	"context"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (p pipeline) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("deleting resource")

	redisInstance := mg.(*exoscalev1.Redis)
	resp, err := p.exo.DeleteDBAASServiceRedis(ctx, redisInstance.GetInstanceName())
	if err != nil {
		return managed.ExternalDelete{}, fmt.Errorf("cannot delete instance: %w", err)
	}
	log.V(1).Info("response", "message", string(resp.Message))
	return managed.ExternalDelete{}, nil
}
