package kafkacontroller

import (
	"context"
	"fmt"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/common"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type connector struct {
	Kube     client.Client
	Recorder event.Recorder
}

// Connect to the exoscale kafka provider.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("connecting resource")

	kafkaInstance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for kafka connector", mg)
	}

	exo, err := pipelineutil.OpenExoscaleClient(ctx, c.Kube, kafkaInstance.GetProviderConfigName(), exoscalesdk.ClientOptWithEndpoint(common.ZoneTranslation[kafkaInstance.Spec.ForProvider.Zone]))
	if err != nil {
		return nil, err
	}
	return newPipeline(c.Kube, c.Recorder, exo.Exoscale), nil

}
