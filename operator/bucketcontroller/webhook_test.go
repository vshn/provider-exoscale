package bucketcontroller

import (
	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBucketValidator_ValidateCreate_RequireProviderConfig(t *testing.T) {
	tests := map[string]struct {
		providerConfig *xpv1.Reference
		expectedError  string
	}{
		"GivenProviderConfig_ThenExpectNoError": {
			providerConfig: &xpv1.Reference{
				Name: "name",
			},
		},
		"GivenNoProviderConfigRef_WhenNoName_ThenExpectError": {
			providerConfig: &xpv1.Reference{
				Name: "",
			},
			expectedError: `.spec.providerConfigRef name is required`,
		},
		"GivenNoProviderConfigRef_WhenNoObject_ThenExpectError": {
			providerConfig: nil,
			expectedError:  `.spec.providerConfigRef name is required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "name", Namespace: "namespace"},
						ProviderConfigReference:          tc.providerConfig,
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateCreate(nil, bucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateCreate_RequireConnectionSecretRef(t *testing.T) {
	tests := map[string]struct {
		secretRef     *xpv1.SecretReference
		expectedError string
	}{
		"GivenWriteConnectionSecretToRef_ThenExpectNoError": {
			secretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		"GivenWriteConnectionSecretToRef_WhenNoName_ThenExpectError": {
			secretRef: &xpv1.SecretReference{
				Namespace: "namespace",
			},
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
		"GivenWriteConnectionSecretToRef_WhenNoNamespace_ThenExpectError": {
			secretRef: &xpv1.SecretReference{
				Name: "name",
			},
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
		"GivenWriteConnectionSecretToRef_WhenObjectIsNil_ThenExpectError": {
			secretRef:     nil,
			expectedError: `.spec.writeConnectionSecretToReference name and namespace are required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						// connection secret is being tested
						WriteConnectionSecretToReference: tc.secretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateCreate(nil, bucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_PreventBucketNameChange(t *testing.T) {
	tests := map[string]struct {
		newBucketName string
		oldBucketName string
		expectedError string
	}{
		"GivenNoNameInStatus_WhenNoNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "",
		},
		"GivenNoNameInStatus_WhenNameInSpec_ThenExpectNil": {
			oldBucketName: "",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecSame_ThenExpectNil": {
			oldBucketName: "my-bucket",
			newBucketName: "my-bucket",
		},
		"GivenNameInStatus_WhenNameInSpecEmpty_ThenExpectNil": {
			oldBucketName: "bucket",
			newBucketName: "", // defaults to metadata.name
		},
		"GivenNameInStatus_WhenNameInSpecDifferent_ThenExpectError": {
			oldBucketName: "my-bucket",
			newBucketName: "different",
			expectedError: `a bucket named "my-bucket" has been created already, you cannot rename it`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider: exoscalev1.BucketParameters{BucketName: tc.oldBucketName},
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: tc.oldBucketName}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider: exoscalev1.BucketParameters{BucketName: tc.newBucketName},
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_RequireProviderConfig(t *testing.T) {
	tests := map[string]struct {
		providerConfigToRef *xpv1.Reference
		expectedError       string
	}{
		"GivenProviderConfigRefWithName_ThenExpectNoError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "provider-config",
			},
		},
		"GivenProviderConfigEmptyRef_ThenExpectError": {
			providerConfigToRef: &xpv1.Reference{
				Name: "",
			},
			expectedError: `.spec.providerConfigRef name is required`,
		},
		"GivenProviderConfigRefNil_ThenExpectError": {
			providerConfigToRef: nil,
			expectedError:       `.spec.providerConfigRef name is required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          tc.providerConfigToRef,
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          tc.providerConfigToRef,
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_RequireConnectionSecretRef(t *testing.T) {
	tests := map[string]struct {
		newConnectionSecretRef *xpv1.SecretReference
		oldConnectionSecretRef *xpv1.SecretReference
		expectedError          string
	}{
		"GivenWriteConnectionSecretToRef_WhenOldIsEqualToNew_ThenExpectNoError": {
			newConnectionSecretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "name",
				Namespace: "namespace",
			},
		},
		"GivenWriteConnectionSecretToRef_WhenOldIsNotEqualToNew_ThenExpectError": {
			newConnectionSecretRef: &xpv1.SecretReference{
				Name:      "new-name",
				Namespace: "new-namespace",
			},
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "old-name",
				Namespace: "old-namespace",
			},
			expectedError: ".spec.writeConnectionSecretToReference name and namespace cannot be changed",
		},
		"GivenWriteConnectionSecretToRef_WhenNewIsNil_ThenExpectError": {
			newConnectionSecretRef: nil,
			oldConnectionSecretRef: &xpv1.SecretReference{
				Name:      "old-name",
				Namespace: "old-namespace",
			},
			expectedError: ".spec.writeConnectionSecretToReference name and namespace cannot be changed",
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: tc.oldConnectionSecretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: tc.newConnectionSecretRef,
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestBucketValidator_ValidateUpdate_PreventZoneChange(t *testing.T) {
	tests := map[string]struct {
		newZone       string
		oldZone       string
		expectedError string
	}{
		"GivenZoneUnchanged_ThenExpectNil": {
			oldZone: "zone",
			newZone: "zone",
		},
		"GivenZoneChanged_ThenExpectError": {
			oldZone:       "zone",
			newZone:       "different",
			expectedError: `a bucket named "bucket" has been created already, you cannot change the zone`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider: exoscalev1.BucketParameters{Zone: tc.oldZone},
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider: exoscalev1.BucketParameters{Zone: tc.newZone},
					ResourceSpec: xpv1.ResourceSpec{
						WriteConnectionSecretToReference: &xpv1.SecretReference{Name: "secret-name", Namespace: "secret-namespace"},
						ProviderConfigReference:          &xpv1.Reference{Name: "provider-config"},
					},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			v := &BucketValidator{log: logr.Discard()}
			err := v.ValidateUpdate(nil, oldBucket, newBucket)
			if tc.expectedError != "" {
				assert.EqualError(t, err, tc.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
