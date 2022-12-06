package kafkacontroller

import (
	"context"
	"errors"
	"fmt"
	"strings"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/google/go-cmp/cmp"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
)

// Observe the external kafka instance.
// Will return wether the the instance exits and if it is up-to-date.
// Observe will also update the status to the observed state and return connection details to connect to the instance.
func (c connection) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalObservation{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	res, err := c.exo.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(instance.GetInstanceName()))
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	external := res.JSON200

	instance.Status.AtProvider, err = getObservation(external)
	if err != nil {
		log.Error(err, "failed to observe kafka instance")
	}

	condition, err := getCondition(external)
	if err != nil {
		log.Error(err, "failed to update kafka condition")
	}
	instance.SetConditions(condition)

	caRes, err := c.exo.GetDbaasCaCertificateWithResponse(ctx)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot retrieve CA certificate: %w", err)
	}
	ca := ""
	if caRes.JSON200 != nil && caRes.JSON200.Certificate != nil {
		ca = *caRes.JSON200.Certificate
	}

	connDetails, err := getConnectionDetails(external, ca)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get kafka connection details: %w", err)
	}

	upToDate, diff := diffParameters(external, instance.Spec.ForProvider)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  upToDate,
		ConnectionDetails: connDetails,
		Diff:              diff,
	}, nil
}

func getObservation(external *oapi.DbaasServiceKafka) (exoscalev1.KafkaObservation, error) {
	notifications, err := mapper.ToNotifications(external.Notifications)
	if err != nil {
		return exoscalev1.KafkaObservation{}, err
	}
	settings, err := mapper.ToRawExtension(external.KafkaSettings)
	if err != nil {
		return exoscalev1.KafkaObservation{}, err
	}

	nodeStates := []exoscalev1.NodeState{}
	if external.NodeStates != nil {
		nodeStates = mapper.ToNodeStates(external.NodeStates)
	}

	restSettings, err := mapper.ToRawExtension(external.KafkaRestSettings)
	if err != nil {
		return exoscalev1.KafkaObservation{}, err
	}

	return exoscalev1.KafkaObservation{
		Version:           pointer.StringDeref(external.Version, ""),
		KafkaSettings:     settings,
		KafkaRestEnabled:  pointer.BoolDeref(external.KafkaRestEnabled, false),
		KafkaRestSettings: restSettings,
		NodeStates:        nodeStates,
		Notifications:     notifications,
	}, nil
}
func getCondition(external *oapi.DbaasServiceKafka) (xpv1.Condition, error) {
	var state oapi.EnumServiceState
	if external.State != nil {
		state = *external.State
	}
	switch state {
	case oapi.EnumServiceStateRunning:
		return exoscalev1.Running(), nil
	case oapi.EnumServiceStateRebuilding:
		return exoscalev1.Rebuilding(), nil
	case oapi.EnumServiceStatePoweroff:
		return exoscalev1.PoweredOff(), nil
	case oapi.EnumServiceStateRebalancing:
		return exoscalev1.Rebalancing(), nil
	default:
		return xpv1.Condition{}, fmt.Errorf("unknown state %q", state)
	}
}
func getConnectionDetails(external *oapi.DbaasServiceKafka, ca string) (map[string][]byte, error) {
	if external.ConnectionInfo == nil {
		return nil, errors.New("no connection details")
	}
	nodes := ""
	if external.ConnectionInfo.Nodes != nil {
		nodes = strings.Join(*external.ConnectionInfo.Nodes, " ")
	}

	if external.ConnectionInfo.AccessCert == nil {
		return nil, errors.New("no certificate returned")
	}
	cert := *external.ConnectionInfo.AccessCert

	if external.ConnectionInfo.AccessKey == nil {
		return nil, errors.New("no key returned")
	}
	key := *external.ConnectionInfo.AccessKey

	if external.Uri == nil {
		return nil, errors.New("no URI returned")
	}
	uri := *external.Uri
	host := ""
	port := ""
	if external.UriParams != nil {
		uriParams := *external.UriParams
		host, _ = uriParams["host"].(string)
		port, _ = uriParams["port"].(string)
	}

	details := map[string][]byte{
		"KAFKA_URI":    []byte(uri),
		"KAFKA_HOST":   []byte(host),
		"KAFKA_PORT":   []byte(port),
		"KAFKA_NODES":  []byte(nodes),
		"service.cert": []byte(cert),
		"service.key":  []byte(key),
		"ca.crt":       []byte(ca),
	}

	if external.KafkaRestEnabled != nil && *external.KafkaRestEnabled && external.ConnectionInfo.RestUri != nil {
		details["KAFKA_REST_URI"] = []byte(*external.ConnectionInfo.RestUri)
	}

	return details, nil
}

func diffParameters(external *oapi.DbaasServiceKafka, expected exoscalev1.KafkaParameters) (bool, string) {
	actualIPFilter := []string{}
	if external.IpFilter != nil {
		actualIPFilter = *external.IpFilter
	}

	actualKafkaSettings, err := mapper.ToRawExtension(external.KafkaSettings)
	if err != nil {
		return false, err.Error()
	}

	actualKafkaRestSettings, err := getActualKafkaRestSettings(external.KafkaRestSettings, expected.KafkaRestSettings)
	if err != nil {
		return false, err.Error()
	}

	actual := exoscalev1.KafkaParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: external.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(external.Maintenance.Time),
		},
		Zone: expected.Zone,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: pointer.BoolDeref(external.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: external.Plan,
			},
			IPFilter: actualIPFilter,
		},
		Version:           expected.Version, // We should never mark somthing as out of date if the versions don't match as update can't modify the version anyway
		KafkaSettings:     actualKafkaSettings,
		KafkaRestEnabled:  pointer.BoolDeref(external.KafkaRestEnabled, false),
		KafkaRestSettings: actualKafkaRestSettings,
	}
	settingComparer := cmp.Comparer(mapper.CompareSettings)
	return cmp.Equal(expected, actual, settingComparer), cmp.Diff(expected, actual, settingComparer)
}

// getActualKafkaRestSettings reads the Kafa REST settings and strips out all non relevant default settings
// Exoscale always returns all defaults, not just the fields we set, so we need to strip them so that we can compare the actual and expected setting.
func getActualKafkaRestSettings(actual *map[string]interface{}, expected runtime.RawExtension) (runtime.RawExtension, error) {
	if actual == nil {
		return runtime.RawExtension{}, nil
	}
	expectedMap, err := mapper.ToMap(expected)
	if err != nil {
		return runtime.RawExtension{}, err
	}
	s := stripRestSettingsDefaults(*actual, expectedMap)
	return mapper.ToRawExtension(&s)
}

// defaultRestSettings are the default settings for Kafka REST.
var defaultRestSettings = map[string]interface{}{
	"consumer_enable_auto_commit":  true,
	"producer_acks":                "1",               // Yes, that's a "1" as a string. I don't know why, that's just how it is..
	"consumer_request_max_bytes":   float64(67108864), // When parsing json into map[string]interface{} we get floats.
	"simpleconsumer_pool_size_max": float64(25),
	"producer_linger_ms":           float64(0),
	"consumer_request_timeout_ms":  float64(1000),
}

func stripRestSettingsDefaults(actual map[string]interface{}, expected map[string]interface{}) map[string]interface{} {
	res := map[string]interface{}{}
	for k, v := range actual {
		d, isDefault := defaultRestSettings[k]
		_, isExpected := expected[k]
		if !isDefault || d != v || isExpected {
			res[k] = v
		}
	}
	return res
}
