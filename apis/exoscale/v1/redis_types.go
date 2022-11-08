package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// RedisParameters are the configurable fields of a Redis instance.
type RedisParameters struct {
	Maintenance MaintenanceSpec `json:"maintenance,omitempty"`

	// +kubebuilder:validation:Required

	// Zone is the datacenter identifier in which the instance runs in.
	Zone Zone `json:"zone"`

	DBaaSParameters `json:",inline"`

	// RedisSettings contains additional Redis settings.
	RedisSettings runtime.RawExtension `json:"redisSettings,omitempty"`
}

// RedisSpec defines the desired state of a Redis.
type RedisSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       RedisParameters `json:"forProvider"`
}

// RedisObservation are the observable fields of a Redis.
type RedisObservation struct {
	Version string `json:"version,omitempty"`
	// RedisSettings contains additional Redis settings as set by the provider.
	RedisSettings runtime.RawExtension `json:"redisSettings,omitempty"`

	// State of individual service nodes
	NodeStates []NodeState `json:"nodeStates,omitempty"`

	// Service notifications
	Notifications []Notification `json:"notifications,omitempty"`
}

// RedisStatus represents the observed state of a Redis instance.
type RedisStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          RedisObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="External Name",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="Plan",type="string",JSONPath=".spec.forProvider.size.plan"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,exoscale}
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-redis,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=redis,versions=v1,name=redis.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// Redis is the API for creating Redis.
type Redis struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RedisSpec   `json:"spec"`
	Status RedisStatus `json:"status,omitempty"`
}

// GetProviderConfigName returns the name of the ProviderConfig.
// Returns empty string if reference not given.
func (in *Redis) GetProviderConfigName() string {
	if ref := in.GetProviderConfigReference(); ref != nil {
		return ref.Name
	}
	return ""
}

// GetInstanceName returns the external name of the instance in the following precedence:
//
//	.metadata.annotations."crossplane.io/external-name"
//	.metadata.name
func (in *Redis) GetInstanceName() string {
	if name := meta.GetExternalName(in); name != "" {
		return name
	}
	return in.Name
}

// +kubebuilder:object:root=true

// RedisList contains a list of Redis
type RedisList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Redis `json:"items"`
}

// Redis type metadata.
var (
	RedisKind             = reflect.TypeOf(Redis{}).Name()
	RedisGroupKind        = schema.GroupKind{Group: Group, Kind: RedisKind}.String()
	RedisKindAPIVersion   = RedisKind + "." + SchemeGroupVersion.String()
	RedisGroupVersionKind = SchemeGroupVersion.WithKind(RedisKind)
)

func init() {
	SchemeBuilder.Register(&Redis{}, &RedisList{})
}
