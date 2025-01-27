package iamkeycontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *IAMKeyPipeline) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("Deleting resource")

	iamKey := fromManaged(mg)
	iamKey.SetConditions(xpv1.Deleting())

	pctx := &pipelineContext{Context: ctx, iamKey: iamKey}
	pipe := pipeline.NewPipeline[*pipelineContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("delete IAM key", p.deleteIAMKey),
			pipe.NewStep("emit event", p.emitDeletionEvent),
		)
	err := pipe.RunWithContext(pctx)
	return managed.ExternalDelete{}, errors.Wrap(err, "cannot deprovision iam key")
}

// deleteIAMKey deletes the IAM key from the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) deleteIAMKey(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	iamKey := ctx.iamKey

	log.Info("Starting IAM key deletion", "keyName", iamKey.Spec.ForProvider.KeyName)

	op, err := p.exoscaleClient.DeleteAPIKey(ctx, iamKey.Status.AtProvider.KeyID)
	if err != nil || op.State != exoscalesdk.OperationStateSuccess {
		return err
	}
	op, err = p.exoscaleClient.DeleteIAMRole(ctx, iamKey.Status.AtProvider.RoleID)
	if err != nil || op.State != exoscalesdk.OperationStateSuccess {
		return err
	}

	return nil
}

func (p *IAMKeyPipeline) emitDeletionEvent(ctx *pipelineContext) error {
	p.recorder.Event(ctx.iamKey, event.Event{
		Type:    event.TypeNormal,
		Reason:  "Deleted",
		Message: "IAMKey deleted",
	})
	return nil
}
