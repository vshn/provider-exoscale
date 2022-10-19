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

	// +kubebuilder:validation:Enum=ch-gva-2;ch-dk-2;de-fra-1;de-muc-1;at-vie-1;bg-sof-1
	// +kubebuilder:validation:Required

	// Zone is the datacenter identifier in which the instance runs in.
	Zone string `json:"zone"`

	DBaaSParameters `json:",inline"`

	// PGSettings contains additional PostgreSQL settings.
	PGSettings runtime.RawExtension `json:"pgSettings,omitempty"`
}

// SizeSpec contains settings to control the sizing of a service.
type SizeSpec struct {
	Plan string `json:"plan,omitempty"`
}

// IPFilter is a list of allowed IPv4 CIDR ranges that can access the service.
// If no IP Filter is set, you may not be able to reach the service.
// A value of `0.0.0.0/0` will open the service to all addresses on the public internet.
type IPFilter []string

// PostgreSQLSpec defines the desired state of a PostgreSQL.
type PostgreSQLSpec struct {
	xpv1.ResourceSpec `json:",inline"`
	ForProvider       PostgreSQLParameters `json:"forProvider"`
}

// PostgreSQLObservation are the observable fields of a PostgreSQL.
type PostgreSQLObservation struct {
	DBaaSParameters `json:",inline"`
	Maintenance     MaintenanceSpec      `json:"maintenance,omitempty"`
	Backup          BackupSpec           `json:"backup,omitempty"`
	NoteStates      []NodeState          `json:"noteStates,omitempty"`
	PGSettings      runtime.RawExtension `json:"pgSettings,omitempty"`
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
