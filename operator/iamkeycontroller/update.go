package iamkeycontroller

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Update implements managed.ExternalClient.
// exoscale.com does not allow any updates on IAM keys.
func (p *IAMKeyPipeline) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Updating resource")
	user := fromManaged(mg)

	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.If(hasSecretRef(user),
				pipeline.NewStepFromFunc("ensure credentials secret", p.ensureCredentialsSecretFn(user)),
			),
			pipeline.NewStepFromFunc("emit event", p.emitUpdateEventFn(user)),
		)
	result := pipe.RunWithContext(ctx)

	return managed.ExternalUpdate{}, errors.Wrap(result.Err(), "cannot update IAM key")
}

func (p *IAMKeyPipeline) emitUpdateEventFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(_ context.Context) error {
		p.recorder.Event(iamKey, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Updated",
			Message: "IAMKey updated",
		})
		return nil
	}
}
