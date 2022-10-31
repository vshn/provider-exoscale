package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// MySQLParameters are the configurable fields of a MySQL.
type MySQLParameters struct {
	Maintenance MaintenanceSpec `json:"maintenance,omitempty"`
	Backup      BackupSpec      `json:"backup,omitempty"`

	// +kubebuilder:validation:Enum=ch-gva-2;ch-dk-2;de-fra-1;de-muc-1;at-vie-1;bg-sof-1
	// +kubebuilder:validation:Required

	// Zone is the datacenter identifier in which the instance runs in.
	Zone string `json:"zone"`

	DBaaSParameters `json:",inline"`

	// MySQLSettings contains additional MySQL settings.
	MySQLSettings runtime.RawExtension `json:"mySQLSettings,omitempty"`
}

// MySQLSpec defines the desired state of a MySQL.
type MySQLSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       MySQLParameters `json:"forProvider"`
}

// MySQLObservation are the observable fields of a MySQL.
type MySQLObservation struct {
	DBaaSParameters `json:",inline"`
	Maintenance     MaintenanceSpec      `json:"maintenance,omitempty"`
	Backup          BackupSpec           `json:"backup,omitempty"`
	NodeStates      []NodeState          `json:"nodeStates,omitempty"`
	MySQLSettings   runtime.RawExtension `json:"mySQLSettings,omitempty"`
}

// MySQLStatus represents the observed state of a MySQL.
type MySQLStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          MySQLObservation `json:"atProvider,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="State",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Synced",type="string",JSONPath=".status.conditions[?(@.type=='Synced')].status"
// +kubebuilder:printcolumn:name="External Name",type="string",JSONPath=".metadata.annotations.crossplane\\.io/external-name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories={crossplane,exoscale}
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-mysql,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=mysqls,versions=v1,name=mysqls.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// MySQL is the API for creating MySQL.
type MySQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MySQLSpec   `json:"spec"`
	Status MySQLStatus `json:"status,omitempty"`
}

// GetProviderConfigName returns the name of the ProviderConfig.
// Returns empty string if reference not given.
func (in *MySQL) GetProviderConfigName() string {
	if ref := in.GetProviderConfigReference(); ref != nil {
		return ref.Name
	}
	return ""
}

// GetInstanceName returns the external name of the instance in the following precedence:
//
//	.metadata.annotations."crossplane.io/external-name"
//	.metadata.name
func (in *MySQL) GetInstanceName() string {
	if name := meta.GetExternalName(in); name != "" {
		return name
	}
	return in.Name
}

// +kubebuilder:object:root=true

// MySQLList contains a list of MySQL
type MySQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MySQL `json:"items"`
}

// MySQL type metadata.
var (
	MySQLKind             = reflect.TypeOf(MySQL{}).Name()
	MySQLGroupKind        = schema.GroupKind{Group: Group, Kind: MySQLKind}.String()
	MySQLKindAPIVersion   = MySQLKind + "." + SchemeGroupVersion.String()
	MySQLGroupVersionKind = SchemeGroupVersion.WithKind(MySQLKind)
)

func init() {
	SchemeBuilder.Register(&MySQL{}, &MySQLList{})
}
