package iamkeycontroller

import (
	"context"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Delete implements managed.ExternalClient.
func (p *IAMKeyPipeline) Delete(ctx context.Context, mg resource.Managed) error {
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
	return errors.Wrap(err, "cannot deprovision iam key")
}

// deleteIAMKey deletes the IAM key from the project associated with the API Key and Secret.
func (p *IAMKeyPipeline) deleteIAMKey(ctx *pipelineContext) error {
	log := controllerruntime.LoggerFrom(ctx)
	iamKey := ctx.iamKey

	log.Info("Starting IAM key deletion", "keyName", iamKey.Spec.ForProvider.KeyName)

	if iamKey.Status.AtProvider.RoleID != "" {

		_, err := ExecuteRequest(ctx, "DELETE", ctx.iamKey.Spec.ForProvider.Zone, "/v2/api-key/"+iamKey.Status.AtProvider.KeyID, p.apiKey, p.apiSecret, nil)
		if err != nil {
			log.Error(err, "Cannot delete apiKey", "keyName", iamKey.Status.AtProvider.KeyID)
			return err
		}
		log.Info("Iam key deleted successfully", "keyName", ctx.iamKey.Spec.ForProvider.KeyName)

		_, err = ExecuteRequest(ctx, "DELETE", ctx.iamKey.Spec.ForProvider.Zone, "/v2/iam-role/"+iamKey.Status.AtProvider.RoleID, p.apiKey, p.apiSecret, nil)
		if err != nil {
			log.Error(err, "Cannot delete iamRole", "iamrole", iamKey.Status.AtProvider.RoleID)
			return err
		}
		log.Info("Iam role deleted successfully", "roleID", iamKey.Status.AtProvider.RoleID)

	} else {
		err := p.exoscaleClient.RevokeIAMAccessKey(ctx, iamKey.Spec.ForProvider.Zone, &exoscalesdk.IAMAccessKey{
			Key: &iamKey.Status.AtProvider.KeyID,
		})
		if err != nil {
			return err
		}
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
