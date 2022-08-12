package v1

const (
	// ExoscaleAPIKey identifies the key in which the API Key of the exoscale.com is expected in a Secret.
	ExoscaleAPIKey = "EXOSCALE_API_KEY"
	// ExoscaleAPISecret identifies the secret in which the API Secret of the exoscale.com is expected in a Secret.
	ExoscaleAPISecret = "EXOSCALE_API_SECRET"
)

type ProviderConfigKey struct{}
type ApiK8sSecretKey struct{}
type MinioClientKey struct{}
type ExoscaleClientKey struct{}
type APIKeyKey struct{}
type APISecretKey struct{}
type EndpointKey struct{}
type ProviderConfigNameKey struct{}
