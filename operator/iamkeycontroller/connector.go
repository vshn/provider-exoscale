package iamkeycontroller

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	"github.com/vshn/provider-exoscale/operator/steps"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type IAMKeyConnector struct {
	kube     client.Client
	recorder event.Recorder
}

type providerConfigKey struct{}
type apiK8sSecretKey struct{}
type exoscaleClientKey struct{}

// Connect implements managed.ExternalConnector.
func (c *IAMKeyConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	ctx = pipeline.MutableContext(ctx)
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	iamKey := fromManaged(mg)
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.fetchProviderConfigFn(*iamKey)),
		pipeline.NewStepFromFunc("fetch API secret", c.fetchSecret),
		pipeline.NewStepFromFunc("read API secret", c.readApiKeyAndSecret),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return nil, result.Err()
	}
	exoscaleClient := pipeline.MustLoadFromContext(ctx, exoscaleClientKey{}).(*exoscalesdk.Client)
	return NewPipeline(c.kube, c.recorder, exoscaleClient), nil
}

func (c *IAMKeyConnector) fetchProviderConfigFn(iamKey exoscalev1.IAMKey) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		providerConfigName := iamKey.GetProviderConfigName()
		if providerConfigName == "" {
			return fmt.Errorf(".spec.providerConfigRef.Name is required")
		}

		providerConfig := &providerv1.ProviderConfig{}
		pipeline.StoreInContext(ctx, providerConfigKey{}, providerConfig)
		err := c.kube.Get(ctx, types.NamespacedName{Name: providerConfigName}, providerConfig)
		return errors.Wrap(err, "cannot get ProviderConfig")
	}
}

func (c *IAMKeyConnector) fetchSecret(ctx context.Context) error {
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

func (c *IAMKeyConnector) readApiKeyAndSecret(ctx context.Context) error {
	secret := pipeline.MustLoadFromContext(ctx, apiK8sSecretKey{}).(*corev1.Secret)
	apiKey, keyExists := secret.Data[exoscalev1.ExoscaleAPIKey]
	apiSecret, secretExists := secret.Data[exoscalev1.ExoscaleAPISecret]
	if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
		exoscaleClient, err := exoscalesdk.NewClient(string(apiKey), string(apiSecret))
		pipeline.StoreInContext(ctx, exoscaleClientKey{}, exoscaleClient)
		return err
	}
	return fmt.Errorf("%s or %s doesn't exist in secret %s/%s", exoscalev1.ExoscaleAPIKey, exoscalev1.ExoscaleAPISecret, secret.Namespace, secret.Name)
}
