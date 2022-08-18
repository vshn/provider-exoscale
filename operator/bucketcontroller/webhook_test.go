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
		providerName  string
		expectedError string
	}{
		"GivenProviderName_ThenExpectNoError": {
			providerName: "provider-config",
		},
		"GivenNoProviderName_ThenExpectError": {
			providerName:  "",
			expectedError: `.spec.providerConfigRef.name is required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			bucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: &xpv1.Reference{Name: tc.providerName},
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
					ForProvider:  exoscalev1.BucketParameters{BucketName: tc.oldBucketName},
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: tc.oldBucketName}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider:  exoscalev1.BucketParameters{BucketName: tc.newBucketName},
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
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
			expectedError: `.spec.providerConfigRef.name is required`,
		},
		"GivenProviderConfigRefNil_ThenExpectError": {
			providerConfigToRef: nil,
			expectedError:       `.spec.providerConfigRef.name is required`,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			oldBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: tc.providerConfigToRef,
					},
					ForProvider: exoscalev1.BucketParameters{BucketName: "bucket"},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ResourceSpec: xpv1.ResourceSpec{
						ProviderConfigReference: tc.providerConfigToRef,
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
					ForProvider:  exoscalev1.BucketParameters{Zone: tc.oldZone},
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
				},
				Status: exoscalev1.BucketStatus{AtProvider: exoscalev1.BucketObservation{BucketName: "bucket"}},
			}
			newBucket := &exoscalev1.Bucket{
				ObjectMeta: metav1.ObjectMeta{Name: "bucket"},
				Spec: exoscalev1.BucketSpec{
					ForProvider:  exoscalev1.BucketParameters{Zone: tc.newZone},
					ResourceSpec: xpv1.ResourceSpec{ProviderConfigReference: &xpv1.Reference{Name: "provider-config"}},
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
