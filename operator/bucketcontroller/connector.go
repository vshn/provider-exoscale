package bucketcontroller

import (
	"context"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/controllerutil"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	corev1 "k8s.io/api/core/v1"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

type bucketConnector struct {
	controllerutil.GenericConnector
}

type connectContext struct {
	controllerutil.GenericConnectContext
	MinioClient *minio.Client
	EndpointURL string
}

// Connect implements managed.ExternalConnecter.
func (c *bucketConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	bucket := fromManaged(mg)

	if isBucketAlreadyDeleted(bucket) {
		log.V(1).Info("Bucket already deleted, skipping observation")
		return &NoopClient{}, nil
	}

	pctx := &connectContext{
		GenericConnectContext: controllerutil.GenericConnectContext{
			Context:            ctx,
			ProviderConfigName: bucket.GetProviderConfigName(),
		},
		EndpointURL: bucket.Spec.ForProvider.EndpointURL,
	}
	pipe := pipeline.NewPipeline[*controllerutil.GenericConnectContext]()
	pipe.WithBeforeHooks(pipelineutil.DebugLogger(&pctx.GenericConnectContext)).
		WithSteps(
			pipe.NewStep("fetch provider config", c.FetchProviderConfig),
			pipe.NewStep("fetch secret", c.FetchSecret),
			pipe.NewStep("validate secret", c.ValidateSecret),
			pipe.NewStep("create S3 client", c.createS3ClientFn(pctx)),
		)
	result := pipe.RunWithContext(&pctx.GenericConnectContext)
	if result != nil {
		return nil, result
	}

	return NewProvisioningPipeline(c.Kube, c.Recorder, pctx.MinioClient), nil
}

// createS3ClientFn creates a new client using the S3 credentials from the Secret.
func (c *bucketConnector) createS3ClientFn(ctx *connectContext) func(genericConnectContext *controllerutil.GenericConnectContext) error {
	return func(_ *controllerutil.GenericConnectContext) error {
		parsed, err := url.Parse(ctx.EndpointURL)
		if err != nil {
			return err
		}

		host := parsed.Host
		if parsed.Host == "" {
			host = parsed.Path // if no scheme is given, it's parsed as a path -.-
		}
		ctx.MinioClient, err = minio.New(host, &minio.Options{
			Creds:  credentials.NewStaticV4(ctx.ApiKey, ctx.ApiSecret, ""),
			Secure: isTLSEnabled(parsed),
		})
		return err
	}

}

// isBucketAlreadyDeleted returns true if the status conditions are in a state where one can assume that the deletion of a bucket was successful in a previous reconciliation.
// This is useful to prevent further reconciliation with possibly lost S3 credentials.
func isBucketAlreadyDeleted(bucket *exoscalev1.Bucket) bool {
	readyCond := findCondition(bucket.Status.Conditions, xpv1.TypeReady)
	syncCond := findCondition(bucket.Status.Conditions, xpv1.TypeSynced)

	if readyCond != nil && syncCond != nil {
		// These 4 criteria must be in exactly this combination
		if readyCond.Status == corev1.ConditionFalse &&
			readyCond.Reason == xpv1.ReasonDeleting &&
			syncCond.Status == corev1.ConditionTrue &&
			syncCond.Reason == xpv1.ReasonReconcileSuccess {
			return true
		}
	}
	return false
}

func findCondition(conds []xpv1.Condition, condType xpv1.ConditionType) *xpv1.Condition {
	for _, cond := range conds {
		if cond.Type == condType {
			return &cond
		}
	}
	return nil
}

// isTLSEnabled returns false if the scheme is explicitly set to `http` or `HTTP`
func isTLSEnabled(u *url.URL) bool {
	return !strings.EqualFold(u.Scheme, "http")
}
