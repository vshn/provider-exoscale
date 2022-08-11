package bucketcontroller

import (
	"context"
	"fmt"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	"net/url"
	"strings"

	pipeline "github.com/ccremer/go-command-pipeline"
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type bucketConnector struct {
	kube     client.Client
	recorder event.Recorder
}

type providerConfigKey struct{}
type apiK8sSecretKey struct{}
type minioClientKey struct{}

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

	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.fetchProviderConfigFn(*bucket)),
		pipeline.NewStepFromFunc("fetch API secret", c.fetchSecret),
		pipeline.NewStepFromFunc("read API secret", c.createS3ClientFn(*bucket)),
	).RunWithContext(ctx)

	if result.IsFailed() {
		return nil, result.Err()
	}

	minioClient := pipeline.MustLoadFromContext(ctx, minioClientKey{}).(*minio.Client)
	return NewProvisioningPipeline(c.kube, c.recorder, minioClient), nil
}

func (c *bucketConnector) fetchProviderConfigFn(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		providerConfigName := bucket.GetProviderConfigName()
		if providerConfigName == "" {
			return fmt.Errorf(".spec.providerConfigRef.Name is required")
		}

		providerConfig := &providerv1.ProviderConfig{}
		pipeline.StoreInContext(ctx, providerConfigKey{}, providerConfig)
		err := c.kube.Get(ctx, types.NamespacedName{Name: providerConfigName}, providerConfig)
		return errors.Wrap(err, "cannot get ProviderConfig")
	}
}

func (c *bucketConnector) fetchSecret(ctx context.Context) error {
	providerConfig := pipeline.MustLoadFromContext(ctx, providerConfigKey{}).(*providerv1.ProviderConfig)
	secretRef := providerConfig.Spec.Credentials.APISecretRef
	apiK8sSecret := &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, apiK8sSecret)
	if err != nil {
		return errors.Wrap(err, "cannot get secret with API token")
	}
	pipeline.StoreInContext(ctx, apiK8sSecretKey{}, apiK8sSecret)
	return nil
}

//createS3ClientFn creates a new client using the S3 credentials from the Secret.
func (c *bucketConnector) createS3ClientFn(bucket exoscalev1.Bucket) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		secret := pipeline.MustLoadFromContext(ctx, apiK8sSecretKey{}).(*corev1.Secret)
		apiKey, keyExists := secret.Data[exoscalev1.ExoscaleAPIKey]
		apiSecret, secretExists := secret.Data[exoscalev1.ExoscaleAPISecret]
		if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
			parsed, err := url.Parse(bucket.Spec.ForProvider.EndpointURL)
			if err != nil {
				return err
			}

			host := parsed.Host
			if parsed.Host == "" {
				host = parsed.Path // if no scheme is given, it's parsed as a path -.-
			}
			s3Client, err := minio.New(host, &minio.Options{
				Creds:  credentials.NewStaticV4(string(apiKey), string(apiSecret), ""),
				Secure: isTLSEnabled(parsed),
			})
			pipeline.StoreInContext(ctx, minioClientKey{}, s3Client)
			return err
		}
		return fmt.Errorf("%s or %s doesn't exist in secret %s/%s", exoscalev1.ExoscaleAPIKey, exoscalev1.ExoscaleAPISecret, secret.Namespace, secret.Name)
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
