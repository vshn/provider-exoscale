package iamkeycontroller

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIAMKeyValidator_ValidateCreate_RequireBuckets(t *testing.T) {
	tests := map[string]struct {
		iamKey        exoscalev1.IAMKey
		bucketNames   []string
		expectedError string
	}{
		"GivenIAMKey_WhenNoBuckets_ThenExpectError": {
			bucketNames:   []string{},
			expectedError: "an IAMKey named \"iamkey-with-no-buckets\" should have at least 1 allowed bucket",
		},
		"GivenIAMKey_When2Buckets_ThenExpectNoError": {
			bucketNames:   []string{"bucket.1", "bucket.2"},
			expectedError: "",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			iamKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{Name: "iamkey-with-no-buckets"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								// buckets is being tested
								Buckets: tc.bucketNames,
							},
						},
					},
				},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateCreate(nil, &iamKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateCreate_RequireWriteConnectionSecretToRef(t *testing.T) {
	tests := map[string]struct {
		iamKey                exoscalev1.IAMKey
		connectionSecretToRef *xpv1.SecretReference
		expectedError         string
	}{
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameAndNamespaceExists_ThenExpectNoError": {
			connectionSecretToRef: &xpv1.SecretReference{
				Name:      "secret-name",
				Namespace: "secret-namespace",
			},
			expectedError: "",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefMissing_ThenExpectError": {
			connectionSecretToRef: &xpv1.SecretReference{},
			expectedError:         ".spec.writeConnectionSecretToRef name and namespace are required",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameMissing_ThenExpectError": {
			connectionSecretToRef: &xpv1.SecretReference{
				Namespace: "secret-namespace",
			},
			expectedError: ".spec.writeConnectionSecretToRef name and namespace are required",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNamespaceMissing_ThenExpectError": {
			connectionSecretToRef: &xpv1.SecretReference{
				Name: "secret-name",
			},
			expectedError: ".spec.writeConnectionSecretToRef name and namespace are required",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefIsNil_ThenExpectError": {
			connectionSecretToRef: nil,
			expectedError:         ".spec.writeConnectionSecretToRef name and namespace are required",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			iamKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
						// connection secret to ref is being tested
						WriteConnectionSecretToReference: tc.connectionSecretToRef,
					},
					ForProvider: exoscalev1.IAMKeyParameters{Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateCreate(nil, &iamKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateCreate_RequireProviderConfigToRef(t *testing.T) {
	tests := map[string]struct {
		iamKey              exoscalev1.IAMKey
		providerConfigToRef *xpv1.Reference
		expectedError       string
	}{
		"GivenIAMKey_WhenProviderConfigToRefNamExists_ThenExpectNoError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "provider-config",
			},
			expectedError: "",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefMissing_ThenExpectError": {
			providerConfigToRef: &xpv1.Reference{},
			expectedError:       ".spec.providerConfigRef name is required",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameMissing_ThenExpectError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "",
			},
			expectedError: ".spec.providerConfigRef name is required",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameIsNil_ThenExpectError": {
			providerConfigToRef: nil,
			expectedError:       ".spec.providerConfigRef name is required",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			iamKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						// provider config to ref is being tested
						ProviderConfigReference: tc.providerConfigToRef,
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateCreate(nil, &iamKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateUpdate_RequireProviderConfigToRef(t *testing.T) {
	tests := map[string]struct {
		providerConfigToRef *xpv1.Reference
		expectedError       string
	}{
		"GivenUpdatedIAMKey_WhenProviderConfigToRefNamExists_ThenExpectNoError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "provider-config",
			},
			expectedError: "",
		},
		"GivenUpdatedIAMKey_WhenWriteConnectionSecretToRefMissing_ThenExpectError": {
			providerConfigToRef: &xpv1.Reference{},
			expectedError:       ".spec.providerConfigRef name is required",
		},
		"GivenUpdatedIAMKey_WhenWriteConnectionSecretToRefNameMissing_ThenExpectError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "",
			},
			expectedError: ".spec.providerConfigRef name is required",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			newIAMKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-key",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						// provider config to ref is being tested
						ProviderConfigReference:          tc.providerConfigToRef,
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "name", Namespace: "namespace"},
					},
					ForProvider: exoscalev1.IAMKeyParameters{KeyName: "key-name", Zone: "CH-1", Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
				Status: exoscalev1.IAMKeyStatus{ResourceStatus: xpv1.ResourceStatus{}, AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			oldIAMKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{Name: "iam-key"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "name", Namespace: "namespace"},
					},
					ForProvider: exoscalev1.IAMKeyParameters{KeyName: "key-name", Zone: "CH-1", Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
				Status: exoscalev1.IAMKeyStatus{ResourceStatus: xpv1.ResourceStatus{}, AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateUpdate(nil, &oldIAMKey, &newIAMKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateUpdate_RequireForProviderImmutable(t *testing.T) {
	tests := map[string]struct {
		newIAMKeyParameters exoscalev1.IAMKeyParameters
		oldIAMKeyParameters exoscalev1.IAMKeyParameters
		expectedError       string
	}{
		"GivenIAMKeyWithKeyId_WhenForProviderObjectUpdated_ThenExpectError": {
			newIAMKeyParameters: exoscalev1.IAMKeyParameters{
				KeyName: "new-key-name",
				Zone:    "CH-1",
				Services: exoscalev1.ServicesSpec{
					SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
				},
			},
			oldIAMKeyParameters: exoscalev1.IAMKeyParameters{
				KeyName: "old-key-name",
				Zone:    "CH-2",
				Services: exoscalev1.ServicesSpec{
					SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1", "bucket.2"}},
				},
			},
			expectedError: "an IAMKey named \"key-name\" has been created already, you cannot update it",
		},
		"GivenIAMKeyWithKeyId_WhenForProviderObjectSame_ThenExpectNoError": {
			newIAMKeyParameters: exoscalev1.IAMKeyParameters{
				KeyName: "key-name",
				Zone:    "CH-1",
				Services: exoscalev1.ServicesSpec{
					SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
				},
			},
			oldIAMKeyParameters: exoscalev1.IAMKeyParameters{
				KeyName: "key-name",
				Zone:    "CH-1",
				Services: exoscalev1.ServicesSpec{
					SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
				},
			},
			expectedError: "",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			newIAMKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{Name: "key-name"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
					// for provider is being tested
					ForProvider: tc.newIAMKeyParameters,
				},
				Status: exoscalev1.IAMKeyStatus{AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			oldIAMKey := exoscalev1.IAMKey{ObjectMeta: metav1.ObjectMeta{Name: "key-name"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
					// for provider is being tested
					ForProvider: tc.oldIAMKeyParameters,
				},
				Status: exoscalev1.IAMKeyStatus{AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateUpdate(nil, &oldIAMKey, &newIAMKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateUpdate_RequireConnectionSecretToRefImmutable(t *testing.T) {
	tests := map[string]struct {
		newConnectionSecretToRef xpv1.SecretReference
		oldConnectionSecretToRef xpv1.SecretReference
		expectedError            string
	}{
		"GivenIAMKeyWithKeyId_WhenWriteConnectionSecretToRefUpdated_ThenExpectError": {
			newConnectionSecretToRef: xpv1.SecretReference{
				Name:      "new-name",
				Namespace: "new-namespace",
			},
			oldConnectionSecretToRef: xpv1.SecretReference{
				Name:      "old-name",
				Namespace: "old-namespace",
			},
			expectedError: "an IAMKey named \"key-name\" has been created already, you cannot update the connection secret reference",
		},
		"GivenIAMKeyWithKeyId_WhenWriteConnectionSecretToRefSame_ThenExpectNoError": {
			newConnectionSecretToRef: xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
			oldConnectionSecretToRef: xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
			expectedError: "",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			newIAMKey := exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{Name: "key-name"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
						// write connection to ref is being tested
						WriteConnectionSecretToReference: &tc.newConnectionSecretToRef,
					},
					ForProvider: exoscalev1.IAMKeyParameters{KeyName: "new-key-name", Zone: "CH-1", Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
				Status: exoscalev1.IAMKeyStatus{AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			oldIAMKey := exoscalev1.IAMKey{ObjectMeta: metav1.ObjectMeta{Name: "key-name"},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{Name: "provider-config"},
						// write connection to ref is being tested
						WriteConnectionSecretToReference: &tc.oldConnectionSecretToRef,
					},
					ForProvider: exoscalev1.IAMKeyParameters{KeyName: "new-key-name", Zone: "CH-1", Services: exoscalev1.ServicesSpec{SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}}}},
				},
				Status: exoscalev1.IAMKeyStatus{AtProvider: exoscalev1.IAMKeyObservation{KeyID: "key-id"}},
			}
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateUpdate(nil, &oldIAMKey, &newIAMKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
