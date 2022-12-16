package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// OpenSearchParameters are the configurable fields of a OpenSearch instance.
type OpenSearchParameters struct {
	Maintenance     MaintenanceSpec `json:"maintenance,omitempty"`
	Backup          BackupSpec      `json:"backup,omitempty"`
	DBaaSParameters `json:",inline"`
	// +kubebuilder:validation:Required
	// Zone is the datacenter identifier in which the instance runs in.
	Zone Zone `json:"zone"`
	// majorVersion - supported versions are "1" and "2" (string)
	MajorVersion       string               `json:"majorVersion,omitempty"`
	OpenSearchSettings runtime.RawExtension `json:"openSearchSettings,omitempty"`
}

// OpenSearchSpec defines the desired state of a OpenSearch.
type OpenSearchSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       OpenSearchParameters `json:"forProvider"`
}

// OpenSearchObservation are the observable fields of a OpenSearch.
type OpenSearchObservation struct {
	MajorVersion string `json:"majorVersion,omitempty"`
	// OpenSearchSettings contains additional OpenSearch settings as set by the provider.
	OpenSearchSettings runtime.RawExtension `json:"openSearchSettings,omitempty"`

	// State of individual service nodes
	NodeStates      []NodeState `json:"nodeStates,omitempty"`
	DBaaSParameters `json:",inline"`
	// Service notifications
	Notifications []Notification  `json:"notifications,omitempty"`
	Maintenance   MaintenanceSpec `json:"maintenance"`
}

// OpenSearchStatus represents the observed state of a OpenSearch instance.
type OpenSearchStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          OpenSearchObservation `json:"atProvider,omitempty"`
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
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-opensearch,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=opensearches,versions=v1,name=opensearch.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// OpenSearch is the API for creating OpenSearch.
type OpenSearch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchSpec   `json:"spec"`
	Status OpenSearchStatus `json:"status,omitempty"`
}

// GetInstanceName returns the external name of the instance in the following precedence:
//
//	.metadata.annotations."crossplane.io/external-name"
//	.metadata.name
func (in *OpenSearch) GetInstanceName() string {
	if name := meta.GetExternalName(in); name != "" {
		return name
	}
	return in.Name
}

// +kubebuilder:object:root=true

// OpenSearchList contains a list of OpenSearch
type OpenSearchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearch `json:"items"`
}

// OpenSearch type metadata.
var (
	OpenSearchKind             = reflect.TypeOf(OpenSearch{}).Name()
	OpenSearchGroupKind        = schema.GroupKind{Group: Group, Kind: OpenSearchKind}.String()
	OpenSearchKindAPIVersion   = OpenSearchKind + "." + SchemeGroupVersion.String()
	OpenSearchGroupVersionKind = SchemeGroupVersion.WithKind(OpenSearchKind)
)

func init() {
	SchemeBuilder.Register(&OpenSearch{}, &OpenSearchList{})
}
