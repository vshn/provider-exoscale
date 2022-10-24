package postgresqlcontroller

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type PostgreSQLConnector struct {
	Kube     client.Client
	Recorder event.Recorder
}

// Connect implements managed.ExternalConnecter.
func (c *PostgreSQLConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	pgInstance := fromManaged(mg)

	exo, err := pipelineutil.OpenExoscaleClient(ctx, c.Kube, pgInstance.GetProviderConfigName(), exoscalesdk.ClientOptWithAPIEndpoint(fmt.Sprintf("https://api-%s.exoscale.com", pgInstance.Spec.ForProvider.Zone)))
	if err != nil {
		return nil, err
	}
	return NewPipeline(c.Kube, c.Recorder, exo.Exoscale), nil
}
