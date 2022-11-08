package v1

import (
	"fmt"
	"regexp"
	"strconv"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	exoscaleoapi "github.com/exoscale/egoscale/v2/oapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type DBaaSParameters struct {
	// TerminationProtection protects against termination and powering off.
	TerminationProtection bool `json:"terminationProtection,omitempty"`
	// Size contains the service capacity settings.
	Size SizeSpec `json:"size,omitempty"`

	IPFilter IPFilter `json:"ipFilter,omitempty"`
}

// NodeState describes the state of a service node.
type NodeState struct {
	// Name of the service node
	Name string `json:"name,omitempty"`
	// Role of this node.
	Role exoscaleoapi.DbaasNodeStateRole `json:"role,omitempty"`
	// State of the service node.
	State exoscaleoapi.DbaasNodeStateState `json:"state,omitempty"`
}

// Notification contains a service message.
type Notification struct {
	// Level of the notification.
	Level exoscaleoapi.DbaasServiceNotificationLevel `json:"level,omitempty"`
	// Message contains the notification.
	Message string `json:"message,omitempty"`
	// Type of the notification.
	Type exoscaleoapi.DbaasServiceNotificationType `json:"type,omitempty"`
	// Metadata contains additional data.
	Metadata runtime.RawExtension `json:"metadata,omitempty"`
}

// BackupSpec contains settings to control the backups of an instance.
type BackupSpec struct {
	// TimeOfDay for doing daily backups, in UTC.
	// Format: "hh:mm:ss".
	TimeOfDay TimeOfDay `json:"timeOfDay,omitempty"`
}

// MaintenanceSpec contains settings to control the maintenance of an instance.
type MaintenanceSpec struct {
	// +kubebuilder:validation:Enum=monday;tuesday;wednesday;thursday;friday;saturday;sunday;never

	// DayOfWeek specifies at which weekday the maintenance is held place.
	// Allowed values are [monday, tuesday, wednesday, thursday, friday, saturday, sunday, never]
	DayOfWeek exoscaleoapi.DbaasServiceMaintenanceDow `json:"dayOfWeek,omitempty"`

	// TimeOfDay for installing updates in UTC.
	// Format: "hh:mm:ss".
	TimeOfDay TimeOfDay `json:"timeOfDay,omitempty"`
}

func (ms MaintenanceSpec) Equals(other MaintenanceSpec) bool {
	return ms.DayOfWeek == other.DayOfWeek && ms.TimeOfDay.String() == other.TimeOfDay.String()
}

var timeOfDayRegex = regexp.MustCompile("^([0-1]?[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])$")

// +kubebuilder:validation:Pattern="^([0-1]?[0-9]|2[0-3]):([0-5][0-9]):([0-5][0-9])$"

// TimeOfDay contains a time in the 24hr clock.
// Format: "hh:mm:ss".
type TimeOfDay string

// String implements fmt.Stringer.
func (t TimeOfDay) String() string {
	return string(t)
}

// Parse returns the hour and minute of the string representation.
// Returns errors if the format is invalid.
func (t TimeOfDay) Parse() (hour, minute, second int64, err error) {
	if t.String() == "" {
		return -1, -1, -1, fmt.Errorf("time cannot be empty")
	}
	arr := timeOfDayRegex.FindStringSubmatch(t.String())
	if len(arr) < 3 {
		return -1, -1, -1, fmt.Errorf("invalid format for time of day (hh:mm:ss): %s", t)
	}
	parts := []int64{0, 0, 0}
	for i, part := range arr[1:] {
		parsed, err := strconv.ParseInt(part, 10, 64)
		if err != nil && part != "" {
			return -1, -1, -1, fmt.Errorf("invalid time given for time of day: %w", err)
		}
		parts[i] = parsed
	}
	return parts[0], parts[1], parts[2], nil
}

// SizeSpec contains settings to control the sizing of a service.
type SizeSpec struct {
	Plan string `json:"plan,omitempty"`
}

func (s SizeSpec) Equals(other SizeSpec) bool {
	return s.Plan == other.Plan
}

// IPFilter is a list of allowed IPv4 CIDR ranges that can access the service.
// If no IP Filter is set, you may not be able to reach the service.
// A value of `0.0.0.0/0` will open the service to all addresses on the public internet.
type IPFilter []string

// +kubebuilder:validation:Enum=ch-gva-2;ch-dk-2;de-fra-1;de-muc-1;at-vie-1;bg-sof-1

// Zone is the datacenter identifier in which the instance runs in.
type Zone string

func (z Zone) String() string {
	return string(z)
}

// Rebuilding returns a Ready condition where the service is rebuilding.
func Rebuilding() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Rebuilding",
		Message:            "The service is being provisioned",
		LastTransitionTime: metav1.Now(),
	}
}

// PoweredOff returns a Ready condition where the service is powered off.
func PoweredOff() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "PoweredOff",
		Message:            "The service is powered off",
		LastTransitionTime: metav1.Now(),
	}
}

// Rebalancing returns a Ready condition where the service is rebalancing.
func Rebalancing() xpv1.Condition {
	return xpv1.Condition{
		Type:               xpv1.TypeReady,
		Status:             corev1.ConditionFalse,
		Reason:             "Rebalancing",
		Message:            "The service is being rebalanced",
		LastTransitionTime: metav1.Now(),
	}
}

// Running returns a Ready condition where the service is running.
func Running() xpv1.Condition {
	c := xpv1.Available()
	c.Message = "The service is running"
	return c
}
