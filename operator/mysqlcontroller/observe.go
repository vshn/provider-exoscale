package mysqlcontroller

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	mySQLInstance := mg.(*exoscalev1.MySQL)

	resp, err := p.exo.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(mySQLInstance.GetInstanceName()))
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("cannot observe mySQLInstance: %w", err)
	}

	mysql := *resp.JSON200
	log.V(2).Info("response", "raw", string(resp.Body))
	log.V(1).Info("retrieved mySQLInstance", "state", mysql.State)

	mySQLInstance.Status.AtProvider, err = mapObservation(mysql)
	if err != nil {
		log.Error(err, "cannot map mySQLInstance observation, ignoring")
	}
	var state oapi.EnumServiceState
	if mysql.State != nil {
		state = *mysql.State
	}
	switch state {
	case oapi.EnumServiceStateRunning:
		mySQLInstance.SetConditions(exoscalev1.Running())
	case oapi.EnumServiceStateRebuilding:
		mySQLInstance.SetConditions(exoscalev1.Rebuilding())
	case oapi.EnumServiceStatePoweroff:
		mySQLInstance.SetConditions(exoscalev1.PoweredOff())
	case oapi.EnumServiceStateRebalancing:
		mySQLInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown mySQLInstance state", "state", state)
	}

	caCert, err := p.exo.GetDatabaseCACertificate(ctx, mySQLInstance.Spec.ForProvider.Zone.String())
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot retrieve CA certificate: %w", err)
	}

	connDetails, err := connectionDetails(mysql, caCert)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}

	params, err := mapParameters(mysql, mySQLInstance.Spec.ForProvider.Zone.String())
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse parameters: %w", err)
	}
	currentParams, err := setSettingsDefaults(ctx, p.exo, &mySQLInstance.Spec.ForProvider)
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

func connectionDetails(in oapi.DbaasServiceMysql, ca string) (managed.ConnectionDetails, error) {
	uri := pointer.StringDeref(in.Uri, "")
	// uri may be absent
	if uri == "" {
		if in.ConnectionInfo == nil || in.ConnectionInfo.Uri == nil || len(*in.ConnectionInfo.Uri) == 0 {
			return map[string][]byte{}, nil
		}
		uri = (*in.ConnectionInfo.Uri)[0]
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot parse connection URI: %w", err)
	}
	password, _ := parsed.User.Password()
	return map[string][]byte{
		"MYSQL_USER":     []byte(parsed.User.Username()),
		"MYSQL_PASSWORD": []byte(password),
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

func mapObservation(instance oapi.DbaasServiceMysql) (exoscalev1.MySQLObservation, error) {
	observation := exoscalev1.MySQLObservation{
		Version:    pointer.StringDeref(instance.Version, ""),
		NodeStates: mapper.ToNodeStates(instance.NodeStates),
	}

	settings, err := mapper.ToRawExtension(instance.MysqlSettings)
	if err != nil {
		return observation, fmt.Errorf("mySQLInstance settings: %w", err)
	}
	observation.MySQLSettings = settings

	observation.DBaaSParameters = mapper.ToDBaaSParameters(instance.TerminationProtection, instance.Plan, instance.IpFilter)
	observation.Maintenance = mapper.ToMaintenance(instance.Maintenance)
	observation.Backup = mapper.ToBackupSpec(instance.BackupSchedule)

	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("mySQLInstance notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in oapi.DbaasServiceMysql, zone string) (*exoscalev1.MySQLParameters, error) {
	settings, err := mapper.ToRawExtension(in.MysqlSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot parse mySQLInstance settings: %w", err)
	}
	return &exoscalev1.MySQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Backup:  mapper.ToBackupSpec(in.BackupSchedule),
		Zone:    exoscalev1.Zone(zone),
		Version: *in.Version,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: pointer.BoolDeref(in.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: mapper.ToSlice(in.IpFilter),
		},
		MySQLSettings: settings,
	}, nil
}
