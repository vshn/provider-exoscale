package iamkeycontroller

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/vshn/provider-exoscale/operator/controllerutil"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	ctrl "sigs.k8s.io/controller-runtime"
)

type IAMKeyConnector struct {
	controllerutil.GenericConnector
}

type connectContext struct {
	controllerutil.GenericConnectContext
	exoscaleClient *exoscalesdk.Client
}

// Connect implements managed.ExternalConnector.
func (c *IAMKeyConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	iamKey := fromManaged(mg)

	pctx := &connectContext{
		GenericConnectContext: controllerutil.GenericConnectContext{Context: ctx, ProviderConfigName: iamKey.GetProviderConfigName()},
	}
	pipe := pipeline.NewPipeline[*controllerutil.GenericConnectContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(&pctx.GenericConnectContext)).
		WithSteps(
			pipe.NewStep("fetch provider config", c.FetchProviderConfig),
			pipe.NewStep("fetch secret", c.FetchSecret),
			pipe.NewStep("validate secret", c.ValidateSecret),
			pipe.NewStep("create exoscale client", c.createExoscaleClientFn(pctx)),
		)
	result := pipe.RunWithContext(&pctx.GenericConnectContext)
	if result != nil {
		return nil, result
	}

	return NewPipeline(c.Kube, c.Recorder, pctx.exoscaleClient), nil
}

func (c *IAMKeyConnector) createExoscaleClientFn(ctx *connectContext) func(genericConnectContext *controllerutil.GenericConnectContext) error {
	return func(_ *controllerutil.GenericConnectContext) error {
		var err error
		ctx.exoscaleClient, err = exoscalesdk.NewClient(ctx.ApiKey, ctx.ApiSecret)
		return err
	}
}
