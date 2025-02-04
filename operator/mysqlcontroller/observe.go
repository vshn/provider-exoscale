package mysqlcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	mysql, err := p.exo.GetDBAASServiceMysql(ctx, mySQLInstance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscalesdk.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("cannot observe mySQLInstance: %w", err)
	}

	log.V(1).Info("retrieved mySQLInstance", "state", mysql.State)

	mySQLInstance.Status.AtProvider, err = mapObservation(mysql)
	if err != nil {
		log.Error(err, "cannot map mySQLInstance observation, ignoring")
	}
	var state exoscalesdk.EnumServiceState
	if mysql.State != "" {
		state = mysql.State
	}
	switch state {
	case exoscalesdk.EnumServiceStateRunning:
		mySQLInstance.SetConditions(exoscalev1.Running())
	case exoscalesdk.EnumServiceStateRebuilding:
		mySQLInstance.SetConditions(exoscalev1.Rebuilding())
	case exoscalesdk.EnumServiceStatePoweroff:
		mySQLInstance.SetConditions(exoscalev1.PoweredOff())
	case exoscalesdk.EnumServiceStateRebalancing:
		mySQLInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown mySQLInstance state", "state", state)
	}

	caCert, err := p.exo.GetDBAASCACertificate(ctx)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot retrieve CA certificate: %w", err)
	}

	connDetails, err := connectionDetails(ctx, mysql, caCert.Certificate, p.exo)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}

	params, err := mapParameters(mysql, mySQLInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse parameters: %w", err)
	}
	currentParams, err := setSettingsDefaults(ctx, *p.exo, &mySQLInstance.Spec.ForProvider)
	if err != nil {
		log.Error(err, "unable to set mysql settings schema")
		currentParams = &mySQLInstance.Spec.ForProvider
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(currentParams, params, log),
		ConnectionDetails: connDetails,
	}, nil
}

func connectionDetails(ctx context.Context, in *exoscalesdk.DBAASServiceMysql, ca string, client *exoscalesdk.Client) (managed.ConnectionDetails, error) {
	uri := in.URI
	// uri may be absent
	if uri == "" {
		if in.ConnectionInfo == nil || in.ConnectionInfo.URI == nil || len(in.ConnectionInfo.URI) == 0 {
			return map[string][]byte{}, nil
		}
		uri = in.ConnectionInfo.URI[0]
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot parse connection URI: %w", err)
	}
	password, err := client.RevealDBAASMysqlUserPassword(ctx, string(in.Name), parsed.User.Username())
	if err != nil {
		return nil, fmt.Errorf("cannot reveal password for MySQL instance: %w", err)
	}
	return map[string][]byte{
		"MYSQL_USER":     []byte(parsed.User.Username()),
		"MYSQL_PASSWORD": []byte(password.Password),
		"MYSQL_URL":      []byte(uri),
		"MYSQL_DB":       []byte(strings.TrimPrefix(parsed.Path, "/")),
		"MYSQL_HOST":     []byte(parsed.Hostname()),
		"MYSQL_PORT":     []byte(parsed.Port()),
		"ca.crt":         []byte(ca),
	}, nil
}

func isUpToDate(current, external *exoscalev1.MySQLParameters, log logr.Logger) bool {
	if external == nil {
		return false
	}
	extIPFilter := []string(external.IPFilter)
	hasSameMajorVersion, err := mapper.CompareMajorVersion(current.Version, external.Version)
	if err != nil {
		log.Error(err, "parse mySQLInstance version", "current", current.Version, "external", external.Version)
	}
	checks := map[string]bool{
		"Maintenance":           current.Maintenance.Equals(external.Maintenance),
		"Backup":                current.Backup.TimeOfDay == external.Backup.TimeOfDay,
		"Zone":                  current.Zone == external.Zone,
		"Version":               hasSameMajorVersion,
		"IPFilter":              mapper.IsSameStringSet(current.IPFilter, &extIPFilter),
		"Size":                  current.Size.Equals(external.Size),
		"TerminationProtection": current.TerminationProtection == external.TerminationProtection,
		"MySQLSettings":         mapper.CompareSettings(current.MySQLSettings, external.MySQLSettings),
	}
	ok := true
	for _, v := range checks {
		if !v {
			log.V(2).Info("mySQLInstance not up-to-date", "check", v)
			ok = false
		}
	}
	return ok
}

func mapObservation(instance *exoscalesdk.DBAASServiceMysql) (exoscalev1.MySQLObservation, error) {

	jsonSettings, err := json.Marshal(instance.MysqlSettings)
	if err != nil {
		return exoscalev1.MySQLObservation{}, fmt.Errorf("error parsing MysqlSettings")
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	nodeStates := []exoscalev1.NodeState{}
	if instance.NodeStates != nil {
		nodeStates = mapper.ToNodeStates(&instance.NodeStates)
	}
	observation := exoscalev1.MySQLObservation{
		Version:    instance.Version,
		NodeStates: nodeStates,
	}

	observation.MySQLSettings = settings

	observation.DBaaSParameters = mapper.ToDBaaSParameters(instance.TerminationProtection, instance.Plan, &instance.IPFilter)
	observation.Maintenance = mapper.ToMaintenance(instance.Maintenance)
	observation.Backup = toBackupSpec(instance.BackupSchedule)

	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("mySQLInstance notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in *exoscalesdk.DBAASServiceMysql, zone exoscalev1.Zone) (*exoscalev1.MySQLParameters, error) {
	jsonSettings, err := json.Marshal(in.MysqlSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot parse mysqlInstance settings: %w", err)
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	return &exoscalev1.MySQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Backup:  toBackupSpec(in.BackupSchedule),
		Zone:    zone,
		Version: in.Version,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: *in.TerminationProtection,
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: in.IPFilter,
		},
		MySQLSettings: settings,
	}, nil
}

func toBackupSpec(schedule *exoscalesdk.DBAASServiceMysqlBackupSchedule) exoscalev1.BackupSpec {
	if schedule == nil {
		return exoscalev1.BackupSpec{}
	}
	hour, min := schedule.BackupHour, schedule.BackupMinute
	return exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay(fmt.Sprintf("%02d:%02d:00", hour, min))}
}
