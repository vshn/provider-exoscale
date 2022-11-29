package kafkacontroller

import (
	"context"
	"errors"
	"fmt"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete idempotently deletes a kafka instance.
// It will not return a "not found" error.
func (c connection) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("deleting resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}
	resp, err := c.exo.DeleteDbaasServiceWithResponse(ctx, instance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrNotFound) {
			return nil
		}
		return fmt.Errorf("cannot delete kafak instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return nil
}
