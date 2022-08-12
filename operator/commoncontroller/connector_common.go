package commoncontroller

import (
	"context"
	"fmt"
	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type GenericConnector struct {
	Kube     client.Client
	Recorder event.Recorder
}

func (c *GenericConnector) FetchProviderConfig(ctx context.Context) error {
	providerConfigName := pipeline.MustLoadFromContext(ctx, exoscalev1.ProviderConfigNameKey{}).(string)
	if providerConfigName == "" {
		return fmt.Errorf(".spec.providerConfigRef.Name is required")
	}

	providerConfig := &providerv1.ProviderConfig{}
	pipeline.StoreInContext(ctx, exoscalev1.ProviderConfigKey{}, providerConfig)
	err := c.Kube.Get(ctx, types.NamespacedName{Name: providerConfigName}, providerConfig)
	return errors.Wrap(err, "cannot get ProviderConfig")

}

func (c *GenericConnector) FetchSecret(ctx context.Context) error {
	providerConfig := pipeline.MustLoadFromContext(ctx, exoscalev1.ProviderConfigKey{}).(*providerv1.ProviderConfig)
	secretRef := providerConfig.Spec.Credentials.APISecretRef
	apiK8sSecret := &corev1.Secret{}
	err := c.Kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, apiK8sSecret)
	if err != nil {
		return errors.Wrap(err, "cannot get secret with API token")
	}
	pipeline.StoreInContext(ctx, exoscalev1.ApiK8sSecretKey{}, apiK8sSecret)
	return nil
}

func (c *GenericConnector) ValidateSecret(ctx context.Context) error {
	secret := pipeline.MustLoadFromContext(ctx, exoscalev1.ApiK8sSecretKey{}).(*corev1.Secret)
	apiKey, keyExists := secret.Data[exoscalev1.ExoscaleAPIKey]
	apiSecret, secretExists := secret.Data[exoscalev1.ExoscaleAPISecret]
	if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
		pipeline.StoreInContext(ctx, exoscalev1.APIKeyKey{}, string(apiKey))
		pipeline.StoreInContext(ctx, exoscalev1.APISecretKey{}, string(apiSecret))
		return nil
	}
	return fmt.Errorf("%s or %s doesn't exist in secret %s/%s", exoscalev1.ExoscaleAPIKey, exoscalev1.ExoscaleAPISecret, secret.Namespace, secret.Name)
}
