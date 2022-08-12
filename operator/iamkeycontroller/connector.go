package iamkeycontroller

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/commoncontroller"
	"github.com/vshn/provider-exoscale/operator/steps"
	ctrl "sigs.k8s.io/controller-runtime"
)

type IAMKeyConnector struct {
	commoncontroller.GenericConnector
}

// Connect implements managed.ExternalConnector.
func (c *IAMKeyConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	iamKey := fromManaged(mg)
	pipeline.StoreInContext(ctx, exoscalev1.ProviderConfigNameKey{}, iamKey.GetProviderConfigName())
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.FetchProviderConfig),
		pipeline.NewStepFromFunc("fetch API secret", c.FetchSecret),
		pipeline.NewStepFromFunc("fetch API secret", c.ValidateSecret),
		pipeline.NewStepFromFunc("read API secret", c.createExoscaleClient),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return nil, result.Err()
	}
	exoscaleClient := pipeline.MustLoadFromContext(ctx, exoscalev1.ExoscaleClientKey{}).(*exoscalesdk.Client)
	return NewPipeline(c.Kube, c.Recorder, exoscaleClient), nil
}

func (c *IAMKeyConnector) createExoscaleClient(ctx context.Context) error {
	apiKey := pipeline.MustLoadFromContext(ctx, exoscalev1.APIKeyKey{}).(string)
	apiSecret := pipeline.MustLoadFromContext(ctx, exoscalev1.APISecretKey{}).(string)
	exoscaleClient, err := exoscalesdk.NewClient(apiKey, apiSecret)
	pipeline.StoreInContext(ctx, exoscalev1.ExoscaleClientKey{}, exoscaleClient)
	return err
}
