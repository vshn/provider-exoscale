package postgresqlcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"

	"github.com/go-logr/logr"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Observe implements managed.ExternalClient.
func (p *pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("Observing resource")

	pgInstance := mg.(*exoscalev1.PostgreSQL)

	pg, err := p.exo.GetDBAASServicePG(ctx, pgInstance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscalesdk.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("cannot observe pgInstance: %w", err)
	}

	log.V(1).Info("Retrieved instance", "state", pg.State)

	pgInstance.Status.AtProvider, err = mapObservation(pg)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot parse instance status")
	}

	setConditionFromState(*pg, pgInstance)

	caCert, err := p.exo.GetDBAASCACertificate(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot retrieve CA certificate")
	}

	params, err := mapParameters(pg, pgInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	connDetails, err := connectionDetails(ctx, pg, caCert.Certificate, p.exo)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot read connection details")
	}

	currentParams, err := setSettingsDefaults(ctx, *p.exo, &pgInstance.Spec.ForProvider)
	if err != nil {
		log.Error(err, "unable to set postgres settings schema")
		currentParams = &pgInstance.Spec.ForProvider
	}
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(currentParams, params, log),
		ConnectionDetails: connDetails,
	}, nil
}

// mapParameters converts a exoscalesdk.DBAASServicePG to the internal exoscalev1.PostgreSQLParameters type.
func mapParameters(in *exoscalesdk.DBAASServicePG, zone exoscalev1.Zone) (*exoscalev1.PostgreSQLParameters, error) {

	jsonSettings, err := json.Marshal(in.PGSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot parse pgInstance settings: %w", err)
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	return &exoscalev1.PostgreSQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Backup: toBackupSpec(in.BackupSchedule),
		Zone:   zone,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: ptr.Deref(in.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: in.IPFilter,
		},
		Version:    in.Version,
		PGSettings: settings,
	}, nil
}

func setConditionFromState(pgExo exoscalesdk.DBAASServicePG, pgInstance *exoscalev1.PostgreSQL) {
	switch pgExo.State {
	case exoscalesdk.EnumServiceStateRunning:
		pgInstance.SetConditions(exoscalev1.Running())
	case exoscalesdk.EnumServiceStateRebuilding:
		pgInstance.SetConditions(exoscalev1.Rebuilding())
	case exoscalesdk.EnumServiceStatePoweroff:
		pgInstance.SetConditions(exoscalev1.PoweredOff())
	case exoscalesdk.EnumServiceStateRebalancing:
		pgInstance.SetConditions(exoscalev1.Rebalancing())
	}
}

var variantAiven = exoscalesdk.EnumPGVariantAiven

// mapObservation fills the status fields from the given response body.
func mapObservation(instance *exoscalesdk.DBAASServicePG) (exoscalev1.PostgreSQLObservation, error) {
	jsonSettings, err := json.Marshal(instance.PGSettings)
	if err != nil {
		return exoscalev1.PostgreSQLObservation{}, fmt.Errorf("error parsing PgSettings")
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	observation := exoscalev1.PostgreSQLObservation{
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: ptr.Deref(instance.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: instance.Plan,
			},
			IPFilter: instance.IPFilter,
		},
		Version: instance.Version,
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: instance.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(instance.Maintenance.Time),
		},
		Backup:     toBackupSpec(instance.BackupSchedule),
		NodeStates: mapper.ToNodeStates(&instance.NodeStates),
	}

	observation.PGSettings = settings

	return observation, nil
}

// isUpToDate returns true if the observed response body matches the desired spec.
func isUpToDate(current, external *exoscalev1.PostgreSQLParameters, log logr.Logger) bool {
	if external == nil {
		return false
	}
	sameMajorVersion, err := mapper.CompareMajorVersion(current.Version, external.Version)
	if err != nil {
		log.Error(err, "parse PostgreSQL version", "current", current.Version, "external", external.Version)
	}
	extIPFilter := []string(external.IPFilter)
	checks := map[string]bool{
		"IPFilter":              mapper.IsSameStringSet(current.IPFilter, &extIPFilter),
		"MajorVersion":          sameMajorVersion,
		"Maintenance":           current.Maintenance.Equals(external.Maintenance),
		"BackupSchedule":        current.Backup.Equals(external.Backup),
		"Size":                  current.Size.Equals(external.Size),
		"TerminationProtection": current.TerminationProtection == external.TerminationProtection,
		"PGSettings":            mapper.CompareSettings(current.PGSettings, external.PGSettings),
	}
	ok := true
	for _, v := range checks {
		if !v {
			log.V(2).Info("instance not up-to-date", "check", v)
			ok = false
		}
	}
	return ok
}

// connectionDetails parses the connection details from the given observation.
func connectionDetails(ctx context.Context, in *exoscalesdk.DBAASServicePG, ca string, client *exoscalesdk.Client) (managed.ConnectionDetails, error) {
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
		return nil, fmt.Errorf("cannot parse connection URL: %w", err)
	}
	password, err := client.RevealDBAASPostgresUserPassword(ctx, string(in.Name), parsed.User.Username())
	if err != nil {
		return nil, fmt.Errorf("cannot reveal password for PostgreSQL instance: %w", err)
	}
	return map[string][]byte{
		"POSTGRESQL_USER":     []byte(parsed.User.Username()),
		"POSTGRESQL_PASSWORD": []byte(password.Password),
		"POSTGRESQL_URL":      []byte(uri),
		"POSTGRESQL_DB":       []byte(strings.TrimPrefix(parsed.Path, "/")),
		"POSTGRESQL_HOST":     []byte(parsed.Hostname()),
		"POSTGRESQL_PORT":     []byte(parsed.Port()),
		"ca.crt":              []byte(ca),
	}, nil
}

func toBackupSpec(schedule *exoscalesdk.DBAASServicePGBackupSchedule) exoscalev1.BackupSpec {
	if schedule == nil {
		return exoscalev1.BackupSpec{}
	}
	hour, min := schedule.BackupHour, schedule.BackupMinute
	return exoscalev1.BackupSpec{TimeOfDay: exoscalev1.TimeOfDay(fmt.Sprintf("%02d:%02d:00", hour, min))}
}
