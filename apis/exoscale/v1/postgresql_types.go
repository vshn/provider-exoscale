package v1

import (
	"reflect"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// PostgreSQLParameters are the configurable fields of a PostgreSQL.
type PostgreSQLParameters struct {
	Maintenance MaintenanceSpec `json:"maintenance,omitempty"`
	Backup      BackupSpec      `json:"backup,omitempty"`

	// +kubebuilder:validation:Required

	// Zone is the datacenter identifier in which the instance runs in.
	Zone Zone `json:"zone"`

	DBaaSParameters `json:",inline"`
	// Version is the (major) version identifier for the instance.
	Version string `json:"version,omitempty"`

	// PGSettings contains additional PostgreSQL settings.
	PGSettings runtime.RawExtension `json:"pgSettings,omitempty"`
}

// PostgreSQLSpec defines the desired state of a PostgreSQL.
type PostgreSQLSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PostgreSQLParameters `json:"forProvider"`
}

// PostgreSQLObservation are the observable fields of a PostgreSQL.
type PostgreSQLObservation struct {
	DBaaSParameters `json:",inline"`
	// Version is the (major) version identifier for the instance.
	Version     string               `json:"version,omitempty"`
	Maintenance MaintenanceSpec      `json:"maintenance,omitempty"`
	Backup      BackupSpec           `json:"backup,omitempty"`
	NodeStates  []NodeState          `json:"nodeStates,omitempty"`
	PGSettings  runtime.RawExtension `json:"pgSettings,omitempty"`
}

// PostgreSQLStatus represents the observed state of a PostgreSQL.
type PostgreSQLStatus struct {
	xpv1.ResourceStatus `json:",inline"`
	AtProvider          PostgreSQLObservation `json:"atProvider,omitempty"`
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
// +kubebuilder:webhook:verbs=create;update,path=/validate-exoscale-crossplane-io-v1-postgresql,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=postgresqls,versions=v1,name=postgresqls.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1

// PostgreSQL is the API for creating PostgreSQL.
type PostgreSQL struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PostgreSQLSpec   `json:"spec"`
	Status PostgreSQLStatus `json:"status,omitempty"`
}

// GetProviderConfigName returns the name of the ProviderConfig.
// Returns empty string if reference not given.
func (in *PostgreSQL) GetProviderConfigName() string {
	if ref := in.GetProviderConfigReference(); ref != nil {
		return ref.Name
	}
	return ""
}

// GetInstanceName returns the external name of the instance in the following precedence:
//
//	.metadata.annotations."crossplane.io/external-name"
//	.metadata.name
func (in *PostgreSQL) GetInstanceName() string {
	if name := meta.GetExternalName(in); name != "" {
		return name
	}
	return in.Name
}

// +kubebuilder:object:root=true

// PostgreSQLList contains a list of PostgreSQL
type PostgreSQLList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgreSQL `json:"items"`
}

// PostgreSQL type metadata.
var (
	PostgreSQLKind             = reflect.TypeOf(PostgreSQL{}).Name()
	PostgreSQLGroupKind        = schema.GroupKind{Group: Group, Kind: PostgreSQLKind}.String()
	PostgreSQLKindAPIVersion   = PostgreSQLKind + "." + SchemeGroupVersion.String()
	PostgreSQLGroupVersionKind = SchemeGroupVersion.WithKind(PostgreSQLKind)
)

func init() {
	SchemeBuilder.Register(&PostgreSQL{}, &PostgreSQLList{})
}
