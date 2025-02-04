package kafkacontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"

	"github.com/google/go-cmp/cmp"
	controllerruntime "sigs.k8s.io/controller-runtime"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
)

// Observe the external kafka instance.
// Will return wether the the instance exits and if it is up-to-date.
// Observe will also update the status to the observed state and return connection details to connect to the instance.
func (p *pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	instance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return managed.ExternalObservation{}, fmt.Errorf("invalid managed resource type %T for kafka connection", mg)
	}

	res, err := p.exo.GetDBAASServiceKafka(ctx, instance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscalesdk.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, err
	}

	instance.Status.AtProvider, err = getObservation(res)
	if err != nil {
		log.Error(err, "failed to observe kafka instance")
	}

	condition, err := getCondition(res)
	if err != nil {
		log.Error(err, "failed to update kafka condition")
	}
	instance.SetConditions(condition)

	caCert, err := p.exo.GetDBAASCACertificate(ctx)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot retrieve CA certificate: %w", err)
	}

	connDetails, err := getConnectionDetails(res, caCert.Certificate)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("failed to get kafka connection details: %w", err)
	}

	currentParams, err := setSettingsDefaults(ctx, *p.exo, &instance.Spec.ForProvider)
	if err != nil {
		log.Error(err, "unable to set kafka settings schema")
		currentParams = &instance.Spec.ForProvider
	}

	upToDate, diff := diffParameters(res, *currentParams)

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  upToDate,
		ConnectionDetails: connDetails,
		Diff:              diff,
	}, nil
}

func getObservation(external *exoscalesdk.DBAASServiceKafka) (exoscalev1.KafkaObservation, error) {
	notifications, err := mapper.ToNotifications(external.Notifications)
	if err != nil {
		return exoscalev1.KafkaObservation{}, fmt.Errorf("error parsing notifications: %w", err)
	}
	jsonSettings, err := json.Marshal(external.KafkaSettings)
	if err != nil {
		return exoscalev1.KafkaObservation{}, fmt.Errorf("error parsing KafkaSettings")
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	nodeStates := []exoscalev1.NodeState{}
	if external.NodeStates != nil {
		nodeStates = mapper.ToNodeStates(&external.NodeStates)
	}

	jsonRestSettings, err := json.Marshal(external.KafkaRestSettings)
	if err != nil {
		return exoscalev1.KafkaObservation{}, fmt.Errorf("error parsing kafka REST settings: %w", err)
	}
	restSettings := runtime.RawExtension{Raw: jsonRestSettings}

	return exoscalev1.KafkaObservation{
		Version:           external.Version,
		KafkaSettings:     settings,
		KafkaRestEnabled:  ptr.Deref(external.KafkaRestEnabled, false),
		KafkaRestSettings: restSettings,
		NodeStates:        nodeStates,
		Notifications:     notifications,
	}, nil
}
func getCondition(external *exoscalesdk.DBAASServiceKafka) (xpv1.Condition, error) {
	var state exoscalesdk.EnumServiceState
	if external.State != "" {
		state = external.State
	}
	switch state {
	case exoscalesdk.EnumServiceStateRunning:
		return exoscalev1.Running(), nil
	case exoscalesdk.EnumServiceStateRebuilding:
		return exoscalev1.Rebuilding(), nil
	case exoscalesdk.EnumServiceStatePoweroff:
		return exoscalev1.PoweredOff(), nil
	case exoscalesdk.EnumServiceStateRebalancing:
		return exoscalev1.Rebalancing(), nil
	default:
		return xpv1.Condition{}, fmt.Errorf("unknown state %q", state)
	}
}
func getConnectionDetails(external *exoscalesdk.DBAASServiceKafka, ca string) (map[string][]byte, error) {
	if external.ConnectionInfo == nil {
		return nil, errors.New("no connection details")
	}
	nodes := ""
	if external.ConnectionInfo.Nodes != nil {
		nodes = strings.Join(external.ConnectionInfo.Nodes, " ")
	}

	if external.ConnectionInfo.AccessCert == "" {
		return nil, errors.New("no certificate returned")
	}
	cert := external.ConnectionInfo.AccessCert

	if external.ConnectionInfo.AccessKey == "" {
		return nil, errors.New("no key returned")
	}
	key := external.ConnectionInfo.AccessKey

	if external.URI == "" {
		return nil, errors.New("no URI returned")
	}
	uri := external.URI
	host := ""
	port := ""
	if external.URIParams != nil {
		uriParams := external.URIParams
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

	if external.KafkaRestEnabled != nil && *external.KafkaRestEnabled && external.ConnectionInfo.RestURI != "" {
		details["KAFKA_REST_URI"] = []byte(external.ConnectionInfo.RestURI)
	}

	return details, nil
}

func diffParameters(external *exoscalesdk.DBAASServiceKafka, expected exoscalev1.KafkaParameters) (bool, string) {
	actualIPFilter := []string{}
	if external.IPFilter != nil {
		actualIPFilter = external.IPFilter
	}

	jsonKafkaSettings, err := json.Marshal(external.KafkaSettings)
	if err != nil {
		return false, err.Error()
	}
	actualKafkaSettings := runtime.RawExtension{Raw: jsonKafkaSettings}

	jsonKafkaRestSettings, err := json.Marshal(external.KafkaRestSettings)
	if err != nil {
		return false, err.Error()
	}
	actualKafkaRestSettings := runtime.RawExtension{Raw: jsonKafkaRestSettings}

	actual := exoscalev1.KafkaParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: external.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(external.Maintenance.Time),
		},
		Zone:              expected.Zone,
		DBaaSParameters:   mapper.ToDBaaSParameters(external.TerminationProtection, external.Plan, &actualIPFilter),
		Version:           expected.Version, // We should never mark somthing as out of date if the versions don't match as update can't modify the version anyway
		KafkaSettings:     actualKafkaSettings,
		KafkaRestEnabled:  ptr.Deref(external.KafkaRestEnabled, false),
		KafkaRestSettings: actualKafkaRestSettings,
	}
	settingComparer := cmp.Comparer(mapper.CompareSettings)
	return cmp.Equal(expected, actual, settingComparer), cmp.Diff(expected, actual, settingComparer)
}
