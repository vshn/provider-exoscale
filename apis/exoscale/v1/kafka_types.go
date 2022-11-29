package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// KafkaParameters are the configurable fields of a Kafka instance.
type KafkaParameters struct {
	Maintenance MaintenanceSpec `json:"maintenance,omitempty"`

	// +kubebuilder:validation:Required

	// Zone is the datacenter identifier in which the instance runs in.
	Zone Zone `json:"zone"`

	DBaaSParameters `json:",inline"`

	// KafkaSettings contains additional Kafka settings.
	KafkaSettings runtime.RawExtension `json:"kafkaSettings,omitempty"`
}

// KafkaSpec defines the desired state of a Kafka.
type KafkaSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       KafkaParameters `json:"forProvider"`
}

// KafkaObservation are the observable fields of a Kafka.
type KafkaObservation struct {
	Version string `json:"version,omitempty"`
	// KafkaSettings contains additional Kafka settings as set by the provider.
	KafkaSettings runtime.RawExtension `json:"kafkaSettings,omitempty"`

	// State of individual service nodes
	NodeStates []NodeState `json:"nodeStates,omitempty"`

	// Service notifications
	Notifications []Notification `json:"notifications,omitempty"`
}

// KafkaStatus represents the observed state of a Kafka instance.
type KafkaStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          KafkaObservation `json:"atProvider,omitempty"`
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
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-kafka,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=kafkas,versions=v1,name=kafkas.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// Kafka is the API for creating Kafka.
type Kafka struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   KafkaSpec   `json:"spec"`
	Status KafkaStatus `json:"status,omitempty"`
}

// GetProviderConfigName returns the name of the ProviderConfig.
// Returns empty string if reference not given.
func (in *Kafka) GetProviderConfigName() string {
	if ref := in.GetProviderConfigReference(); ref != nil {
		return ref.Name
	}
	return ""
}

// GetInstanceName returns the external name of the instance in the following precedence:
//
//	.metadata.annotations."crossplane.io/external-name"
//	.metadata.name
func (in *Kafka) GetInstanceName() string {
	if name := meta.GetExternalName(in); name != "" {
		return name
	}
	return in.Name
}

// +kubebuilder:object:root=true

// KafkaList contains a list of Kafka
type KafkaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Kafka `json:"items"`
}

// Kafka type metadata.
var (
	KafkaKind             = reflect.TypeOf(Kafka{}).Name()
	KafkaGroupKind        = schema.GroupKind{Group: Group, Kind: KafkaKind}.String()
	KafkaKindAPIVersion   = KafkaKind + "." + SchemeGroupVersion.String()
	KafkaGroupVersionKind = SchemeGroupVersion.WithKind(KafkaKind)
)

func init() {
	SchemeBuilder.Register(&Kafka{}, &KafkaList{})
}
