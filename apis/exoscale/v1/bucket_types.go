package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	// DeleteIfEmpty only deletes the bucket if the bucket is empty.
	DeleteIfEmpty BucketDeletionPolicy = "DeleteIfEmpty"
	// DeleteAll recursively deletes all objects in the bucket and then removes it.
	DeleteAll BucketDeletionPolicy = "DeleteAll"
)

// BucketDeletionPolicy determines how buckets should be deleted when a Bucket is deleted.
type BucketDeletionPolicy string

// BucketParameters are the configurable fields of a Bucket.
type BucketParameters struct {

	// BucketName is the name of the bucket to create.
	// Defaults to `metadata.name` if unset.
	// Cannot be changed after bucket is created.
	// Name must be acceptable by the S3 protocol, which follows RFC 1123.
	// Be aware that S3 providers may require a unique name across the platform or zone.
	BucketName string `json:"bucketName,omitempty"`

	// +kubebuilder:validation:Required

	// Deprecated: Only here for compatibility with legacy Bucket objects
	EndpointURL string `json:"endpointURL,omitempty"`

	// +kubebuilder:validation:Required

	// Zone is the name of the zone where the bucket shall be created.
	// The zone must be available in the S3 endpoint.
	// Cannot be changed after bucket is created.
	Zone string `json:"zone"`

	// +kubebuilder:validation:Enum=DeleteIfEmpty;DeleteAll
	// +kubebuilder:default="DeleteIfEmpty"

	// BucketDeletionPolicy determines how buckets should be deleted when Bucket is deleted.
	//  `DeleteIfEmpty` only deletes the bucket if the bucket is empty.
	//  `DeleteAll` recursively deletes all objects in the bucket and then removes it.
	// To skip deletion of the bucket (orphan it) set `spec.deletionPolicy=Orphan`.
	BucketDeletionPolicy BucketDeletionPolicy `json:"bucketDeletionPolicy,omitempty"`
}

// BucketSpec defines the desired state of a Bucket.
type BucketSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       BucketParameters `json:"forProvider"`
}

// BucketObservation are the observable fields of a Bucket.
type BucketObservation struct {
	// BucketName is the name of the actual bucket.
	BucketName string `json:"bucketName,omitempty"`
}

// BucketStatus represents the observed state of a Bucket.
type BucketStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	Endpoint            string            `json:"endpoint,omitempty"`
	EndpointURL         string            `json:"endpointURL,omitempty"`
	AtProvider          BucketObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="External Name",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Endpoint",type="string",JSONPath=".spec.forProvider.endpointURL"
// +kubebuilder:printcolumn:name="Bucket Name",type="string",JSONPath=".status.atProvider.bucketName"
// +kubebuilder:printcolumn:name="Zone",type="string",JSONPath=".spec.forProvider.zone"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,exoscale}
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-bucket,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=buckets,versions=v1,name=buckets.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// Bucket is the API for creating S3 buckets.
type Bucket struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   BucketSpec   `json:"spec"`
	Status BucketStatus `json:"status,omitempty"`
}

// GetBucketName returns the spec.forProvider.bucketName if given, otherwise defaults to metadata.name.
func (in *Bucket) GetBucketName() string {
	if in.Spec.ForProvider.BucketName == "" {
		return in.Name
	}
	return in.Spec.ForProvider.BucketName
}

// GetProviderConfigName returns the name of the ProviderConfig.
// Returns empty string if reference not given.
func (in *Bucket) GetProviderConfigName() string {
	if ref := in.GetProviderConfigReference(); ref != nil {
		return ref.Name
	}
	return ""
}

// +kubebuilder:object:root=true

// BucketList contains a list of Bucket
type BucketList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Bucket `json:"items"`
}

// Bucket type metadata.
var (
	BucketKind             = reflect.TypeOf(Bucket{}).Name()
	BucketGroupKind        = schema.GroupKind{Group: Group, Kind: BucketKind}.String()
	BucketKindAPIVersion   = BucketKind + "." + SchemeGroupVersion.String()
	BucketGroupVersionKind = SchemeGroupVersion.WithKind(BucketKind)
)

func init() {
	SchemeBuilder.Register(&Bucket{}, &BucketList{})
}
