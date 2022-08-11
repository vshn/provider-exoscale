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

func TestIAMKeyValidator_ValidateCreate(t *testing.T) {
	tests := map[string]struct {
		iamKey        exoscalev1.IAMKey
		expectedError string
	}{
		"GivenIAMKey_WhenNoBuckets_ThenExpectError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-no-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{},
					},
				},
			},
			expectedError: "an IAMKey named \"iamkey-with-no-buckets\" should have at least 1 allowed bucket",
		},
		"GivenIAMKey_When2Buckets_ThenExpectNoError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								Buckets: []string{"bucket.1", "bucket.2"},
							},
						},
					},
				},
			},
			expectedError: "",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameAndNamespaceExists_ThenExpectNoError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "secret-name",
							Namespace: "secret-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								Buckets: []string{"bucket.1", "bucket.2"},
							},
						},
					},
				},
			},
			expectedError: "",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefMissing_ThenExpectError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: nil,
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								Buckets: []string{"bucket.1", "bucket.2"},
							},
						},
					},
				},
			},
			expectedError: "an IAMKey named \"iamkey-with-buckets\" requires a connection secret reference with name and namespace",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNameMissing_ThenExpectError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "",
							Namespace: "secret-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								Buckets: []string{"bucket.1", "bucket.2"},
							},
						},
					},
				},
			},
			expectedError: "an IAMKey named \"iamkey-with-buckets\" requires a connection secret reference with name and namespace",
		},
		"GivenIAMKey_WhenWriteConnectionSecretToRefNamespaceMissing_ThenExpectError": {
			iamKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "secret-name",
							Namespace: "",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{
								Buckets: []string{"bucket.1", "bucket.2"},
							},
						},
					},
				},
			},
			expectedError: "an IAMKey named \"iamkey-with-buckets\" requires a connection secret reference with name and namespace",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateCreate(nil, &tc.iamKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestIAMKeyValidator_ValidateUpdate(t *testing.T) {
	tests := map[string]struct {
		newIAMKey     exoscalev1.IAMKey
		oldIAMKey     exoscalev1.IAMKey
		expectedError string
	}{
		"GivenIAMKey_WhenKeyIDInStatus_ThenExpectNoError": {
			newIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iamkey-with-no-buckets",
				},
				Spec: exoscalev1.IAMKeySpec{},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "",
					},
				},
			},
			oldIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "old-key",
				},
				Spec: exoscalev1.IAMKeySpec{
					ForProvider: exoscalev1.IAMKeyParameters{
						Services: exoscalev1.ServicesSpec{},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "",
					},
				},
			},
			expectedError: "",
		},
		"GivenIAMKey_WhenNoUpdateOnSecretRefAndForProvider_ThenExpectError": {
			newIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-key",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "name",
							Namespace: "namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "key-name",
						Zone:    "CH-1",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			oldIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "iam-key",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "name",
							Namespace: "namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "key-name",
						Zone:    "CH-1",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			expectedError: "",
		},
		"GivenIAMKeyWithKeyId_WhenForProviderObjectUpdated_ThenExpectError": {
			newIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-name",
				},
				Spec: exoscalev1.IAMKeySpec{
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "new-key-name",
						Zone:    "CH-1",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			oldIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-name",
				},
				Spec: exoscalev1.IAMKeySpec{
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "old-key-name",
						Zone:    "CH-2",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1", "bucket.2"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			expectedError: "an IAMKey named \"key-name\" has been created already, you cannot update it",
		},
		"GivenIAMKeyWithKeyId_WhenWriteConnectionSecretToRefUpdated_ThenExpectError": {
			newIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-name",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "new-name",
							Namespace: "new-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "key-name",
						Zone:    "CH-1",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			oldIAMKey: exoscalev1.IAMKey{
				ObjectMeta: metav1.ObjectMeta{
					Name: "key-name",
				},
				Spec: exoscalev1.IAMKeySpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{
							Name:      "old-name",
							Namespace: "old-namespace",
						},
					},
					ForProvider: exoscalev1.IAMKeyParameters{
						KeyName: "key-name",
						Zone:    "CH-1",
						Services: exoscalev1.ServicesSpec{
							SOS: exoscalev1.SOSSpec{Buckets: []string{"bucket.1"}},
						},
					},
				},
				Status: exoscalev1.IAMKeyStatus{
					ResourceStatus: xpv1.ResourceStatus{},
					AtProvider: exoscalev1.IAMKeyObservation{
						KeyID: "key-id",
					},
				},
			},
			expectedError: "an IAMKey named \"key-name\" has been created already, you cannot update the connection secret reference",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			validator := &IAMKeyValidator{log: logr.Discard()}
			err := validator.ValidateUpdate(nil, &tc.oldIAMKey, &tc.newIAMKey)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
