package pipelineutil

import (
	"context"
	"fmt"

	pipeline "github.com/ccremer/go-command-pipeline"
	"github.com/crossplane/crossplane-runtime/pkg/errors"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
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

type ExoscaleConnector struct {
	ApiKey    string
	ApiSecret string
	Exoscale  *exoscalesdk.Client
}

type connectContext struct {
	context.Context
	kube               client.Client
	ProviderConfigName string

	providerConfig   *providerv1.ProviderConfig
	credentialSecret *corev1.Secret
	apiKey           string
	apiSecret        string
	exoscaleClient   *exoscalesdk.Client
	opts             []exoscalesdk.ClientOpt
}

// OpenExoscaleClient fetches the ProviderConfig given by the name, fetches the API secret and returns a ExoscaleConnector with an initialized client.
func OpenExoscaleClient(ctx context.Context, kube client.Client, providerConfigRef string, opts ...exoscalesdk.ClientOpt) (*ExoscaleConnector, error) {
	pctx := &connectContext{
		Context:            ctx,
		kube:               kube,
		ProviderConfigName: providerConfigRef,
		opts:               opts,
	}

	pipe := pipeline.NewPipeline[*connectContext]()
	pipe.WithBeforeHooks(DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("fetch provider config", fetchProviderConfig),
			pipe.NewStep("fetch secret", fetchSecret),
			pipe.NewStep("validate secret", validateSecret),
			pipe.NewStep("create exoscale client", createExoscaleClient),
		)
	err := pipe.RunWithContext(pctx)
	if err != nil {
		return nil, err
	}
	return &ExoscaleConnector{
		ApiSecret: pctx.apiSecret,
		ApiKey:    pctx.apiKey,
		Exoscale:  pctx.exoscaleClient,
	}, nil
}

func fetchProviderConfig(ctx *connectContext) error {
	ctx.providerConfig = &providerv1.ProviderConfig{}
	err := ctx.kube.Get(ctx, types.NamespacedName{Name: ctx.ProviderConfigName}, ctx.providerConfig)
	return errors.Wrap(err, "cannot get ProviderConfig")
}

func fetchSecret(ctx *connectContext) error {
	secretRef := ctx.providerConfig.Spec.Credentials.APISecretRef
	ctx.credentialSecret = &corev1.Secret{}

	err := ctx.kube.Get(ctx, types.NamespacedName{Name: secretRef.Name, Namespace: secretRef.Namespace}, ctx.credentialSecret)

	return errors.Wrap(err, "cannot get secret with API token")
}

func validateSecret(ctx *connectContext) error {
	secret := ctx.credentialSecret
	apiKey, keyExists := secret.Data[ExoscaleAPIKey]
	apiSecret, secretExists := secret.Data[ExoscaleAPISecret]
	if (keyExists && secretExists) && (string(apiKey) != "" && string(apiSecret) != "") {
		ctx.apiKey = string(apiKey)
		ctx.apiSecret = string(apiSecret)
		return nil
	}
	return fmt.Errorf("%s or %s doesn't exist in secret %s/%s", ExoscaleAPIKey, ExoscaleAPISecret, secret.Namespace, secret.Name)
}

func createExoscaleClient(ctx *connectContext) error {
	creds := credentials.NewStaticCredentials(ctx.apiKey, ctx.apiSecret)
	ec, err := exoscalesdk.NewClient(creds, ctx.opts...)
	ctx.exoscaleClient = ec
	return err
}

// FetchProviderConfig returns the apiKey and apiSecret of the given providerConfigRef
func FetchProviderConfig(ctx context.Context, kube client.Client, providerConfigRef string) (string, string, error) {
	pctx := &connectContext{
		Context:            ctx,
		kube:               kube,
		ProviderConfigName: providerConfigRef,
	}

	pipe := pipeline.NewPipeline[*connectContext]()
	pipe.WithBeforeHooks(DebugLogger(pctx)).
		WithSteps(
			pipe.NewStep("fetch provider config", fetchProviderConfig),
			pipe.NewStep("fetch secret", fetchSecret),
			pipe.NewStep("validate secret", validateSecret),
		)
	err := pipe.RunWithContext(pctx)
	if err != nil {
		return "", "", err
	}
	return pctx.apiKey, pctx.apiSecret, nil
}
