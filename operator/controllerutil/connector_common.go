package controllerutil

import (
	"context"
	"fmt"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// ExoscaleAPIKey identifies the key in which the API Key of the exoscale.com is expected in a Secret.
	ExoscaleAPIKey = "EXOSCALE_API_KEY"
	// ExoscaleAPISecret identifies the secret in which the API Secret of the exoscale.com is expected in a Secret.
	ExoscaleAPISecret = "EXOSCALE_API_SECRET"
)

type GenericConnector struct {
	Kube     client.Client
	Recorder event.Recorder
}

type GenericConnectContext struct {
	context.Context
	ApiKey             string
	ApiSecret          string
	ProviderConfigName string

	providerConfig   *providerv1.ProviderConfig
	credentialSecret *corev1.Secret
}

func (c *GenericConnector) FetchProviderConfig(ctx *GenericConnectContext) error {
	ctx.providerConfig = &providerv1.ProviderConfig{}
	err := c.Kube.Get(ctx, types.NamespacedName{Name: ctx.ProviderConfigName}, ctx.providerConfig)
	return errors.Wrap(err, "cannot get ProviderConfig")
}

func (c *GenericConnector) FetchSecret(ctx *GenericConnectContext) error {
	secretRef := ctx.providerConfig.Spec.Credentials.APISecretRef
	ctx.credentialSecret = &corev1.Secret{}
	err := c.Kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, ctx.credentialSecret)
	return errors.Wrap(err, "cannot get secret with API token")
}

func (c *GenericConnector) ValidateSecret(ctx *GenericConnectContext) error {
	secret := ctx.credentialSecret
	apiKey, keyExists := secret.Data[ExoscaleAPIKey]
	apiSecret, secretExists := secret.Data[ExoscaleAPISecret]
	if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
		ctx.ApiKey = string(apiKey)
		ctx.ApiSecret = string(apiSecret)
		return nil
	}
	return fmt.Errorf("%s or %s doesn't exist in secret %s/%s", ExoscaleAPIKey, ExoscaleAPISecret, secret.Namespace, secret.Name)
}
