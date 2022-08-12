package bucketcontroller

import (
	"context"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/commoncontroller"
	"github.com/vshn/provider-exoscale/operator/steps"
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
	commoncontroller.GenericConnector
}

// Connect implements managed.ExternalConnecter.
func (c *bucketConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	bucket := fromManaged(mg)

	if isBucketAlreadyDeleted(bucket) {
		// We do this to prevent after-deletion reconciliations since there is a chance that we might not have the access credentials anymore.
		log.V(1).Info("Bucket already deleted, skipping observation")
		return &NoopClient{}, nil
	}

	pipeline.StoreInContext(ctx, exoscalev1.ProviderConfigNameKey{}, bucket.GetProviderConfigName())
	pipeline.StoreInContext(ctx, exoscalev1.EndpointKey{}, bucket.Spec.ForProvider.EndpointURL)
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.FetchProviderConfig),
		pipeline.NewStepFromFunc("fetch API secret", c.FetchSecret),
		pipeline.NewStepFromFunc("fetch API secret", c.ValidateSecret),
		pipeline.NewStepFromFunc("read API secret", c.createS3Client),
	).RunWithContext(ctx)

	if result.IsFailed() {
		return nil, result.Err()
	}

	minioClient := pipeline.MustLoadFromContext(ctx, exoscalev1.MinioClientKey{}).(*minio.Client)
	return NewProvisioningPipeline(c.Kube, c.Recorder, minioClient), nil
}

//createS3Client creates a new client using the S3 credentials from the Secret.
func (c *bucketConnector) createS3Client(ctx context.Context) error {
	apiKey := pipeline.MustLoadFromContext(ctx, exoscalev1.APIKeyKey{}).(string)
	apiSecret := pipeline.MustLoadFromContext(ctx, exoscalev1.APISecretKey{}).(string)
	endpoint := pipeline.MustLoadFromContext(ctx, exoscalev1.EndpointKey{}).(string)

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return err
	}

	host := parsed.Host
	if parsed.Host == "" {
		host = parsed.Path // if no scheme is given, it's parsed as a path -.-
	}
	s3Client, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(apiKey, apiSecret, ""),
		Secure: isTLSEnabled(parsed),
	})
	pipeline.StoreInContext(ctx, exoscalev1.MinioClientKey{}, s3Client)
	return err
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
