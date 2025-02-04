package rediscontroller

import (
	"context"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/common"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type connector struct {
	kube     client.Client
	recorder event.Recorder
}

// Connect implements managed.ExternalConnecter.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("connecting resource")

	redisInstance := mg.(*exoscalev1.Redis)

	exo, err := pipelineutil.OpenExoscaleClient(ctx, c.kube, redisInstance.GetProviderConfigName(), exoscalesdk.ClientOptWithEndpoint(common.ZoneTranslation[redisInstance.Spec.ForProvider.Zone]))
	if err != nil {
		return nil, err
	}
	return newPipeline(c.kube, c.recorder, exo.Exoscale), nil
}
