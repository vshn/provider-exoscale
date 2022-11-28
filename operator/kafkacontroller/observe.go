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
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
)

// Observe implements managed.ExternalClient
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

	upToDate, diff := diffParamters(external, instance.Spec.ForProvider)

	return managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        upToDate,
		ResourceLateInitialized: false,
		ConnectionDetails:       connDetails,
		Diff:                    diff,
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

	return exoscalev1.KafkaObservation{
		Version:       pointer.StringDeref(external.Version, ""),
		KafkaSettings: settings,
		NodeStates:    nodeStates,
		Notifications: notifications,
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

	return map[string][]byte{
		"KAFKA_URI":   []byte(uri),
		"KAFKA_HOST":  []byte(host),
		"KAFKA_PORT":  []byte(port),
		"KAFKA_NODES": []byte(nodes),
		"cert.pem":    []byte(cert),
		"key.pem":     []byte(key),
		"ca.crt":      []byte(ca),
	}, nil
}

func diffParamters(external *oapi.DbaasServiceKafka, expected exoscalev1.KafkaParameters) (bool, string) {
	actualIPFilter := []string{}
	if external.IpFilter != nil {
		actualIPFilter = *external.IpFilter
	}
	actualKafkaSettings, err := mapper.ToRawExtension(external.KafkaRestSettings)
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
		KafkaSettings: actualKafkaSettings,
	}

	return cmp.Equal(expected, actual), cmp.Diff(expected, actual)
}
