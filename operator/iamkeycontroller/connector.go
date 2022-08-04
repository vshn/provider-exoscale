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

const (
	// ExoscaleAPIKey identifies the key in which the API Key of the exoscale.com is expected in a Secret.
	ExoscaleAPIKey = "EXOSCALE_API_KEY"
	// ExoscaleAPISecret identifies the secret in which the API Secret of the exoscale.com is expected in a Secret.
	ExoscaleAPISecret = "EXOSCALE_API_SECRET"
)

type IAMKeyConnector struct {
	kube     client.Client
	recorder event.Recorder

	iamKey         *exoscalev1.IAMKey
	providerConfig *providerv1.ProviderConfig
	apiK8sSecret   *corev1.Secret
	apiKey         string
	apiSecret      string
}

// Connect implements managed.ExternalConnector.
func (c *IAMKeyConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Connecting resource")

	iamKey := fromManaged(mg)
	c.iamKey = iamKey
	result := pipeline.NewPipeline().WithBeforeHooks(steps.DebugLogger(ctx)).WithSteps(
		pipeline.NewStepFromFunc("fetch provider config", c.fetchProviderConfig),
		pipeline.NewStepFromFunc("fetch API secret", c.fetchSecret),
		pipeline.NewStepFromFunc("read API secret", c.readApiKeyAndSecret),
	).RunWithContext(ctx)
	if result.IsFailed() {
		return nil, result.Err()
	}
	exoscaleClient, err := exoscalesdk.NewClient(c.apiKey, c.apiSecret)
	return NewPipeline(c.kube, c.recorder, exoscaleClient), err
}

func (c *IAMKeyConnector) fetchProviderConfig(ctx context.Context) error {
	providerConfigName := c.iamKey.GetProviderConfigName()
	if providerConfigName == "" {
		return fmt.Errorf(".spec.providerConfigRef.Name is required")
	}
	c.providerConfig = &providerv1.ProviderConfig{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: providerConfigName}, c.providerConfig)
	return errors.Wrap(err, "cannot get ProviderConfig")
}

func (c *IAMKeyConnector) fetchSecret(ctx context.Context) error {
	secretRef := c.providerConfig.Spec.Credentials.APISecretRef
	c.apiK8sSecret = &corev1.Secret{}
	err := c.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, c.apiK8sSecret)
	return errors.Wrap(err, "cannot get secret with API token")
}

func (c *IAMKeyConnector) readApiKeyAndSecret(_ context.Context) error {
	secret := c.apiK8sSecret
	apiKey, keyExists := secret.Data[ExoscaleAPIKey]
	apiSecret, secretExists := secret.Data[ExoscaleAPISecret]
	if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
		c.apiKey = string(apiKey)
		c.apiSecret = string(apiSecret)
		return nil
	}
	return fmt.Errorf("%s and %s doesn't exist in secret %s/%s", ExoscaleAPIKey, ExoscaleAPISecret, secret.Namespace, secret.Name)
}
