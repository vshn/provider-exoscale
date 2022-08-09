package iamkeycontroller

import (
	"context"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *IAMKeyPipeline) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	iamKey := fromManaged(mg)
	pipe := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).
		WithSteps(
			pipeline.NewStepFromFunc("delete IAM key", p.deleteIAMKeyFn(iamKey)),
			pipeline.NewStepFromFunc("emit event", p.emitDeletionEventFn(iamKey)),
		)
	result := pipe.RunWithContext(ctx)
	return errors.Wrap(result.Err(), "cannot deprovision IAM key")
}

// deleteIAMKeyFn deletes the IAM key from the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) deleteIAMKeyFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log := controllerruntime.LoggerFrom(ctx)

		err := p.exoscaleClient.RevokeIAMAccessKey(ctx, iamKey.Spec.ForProvider.Zone, &exoscalesdk.IAMAccessKey{
			Key: &iamKey.Status.AtProvider.KeyID,
		})
		if err != nil {
			return err
		}
		log.V(1).Info("Deleted IAM key in exoscale", "keyID", iamKey.Status.AtProvider.KeyID)
		return nil
	}
}

func (p *IAMKeyPipeline) emitDeletionEventFn(iamKey *exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(_ context.Context) error {
		p.recorder.Event(iamKey, event.Event{
			Type:    event.TypeNormal,
			Reason:  "Deleted",
			Message: "IAMKey deleted",
		})
		return nil
	}
}
