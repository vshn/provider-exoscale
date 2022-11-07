package postgresqlcontroller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/errors"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
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
	log.V(1).Info("Observing resource")

	pgInstance := fromManaged(mg)

	resp, err := p.exo.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(pgInstance.GetInstanceName()))
	if err != nil {
		return managed.ExternalObservation{}, ignoreNotFound(err)
	}
	pgExo := *resp.JSON200
	log.V(2).Info("Response", "raw", resp.JSON200)
	log.V(1).Info("Retrieved instance", "state", pgExo.State)

	pgInstance.Status.AtProvider, err = mapObservation(pgExo)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot parse instance status")
	}

	setConditionFromState(pgExo, pgInstance)

	ca, err := p.exo.GetDatabaseCACertificate(ctx, pgInstance.Spec.ForProvider.Zone.String())
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot retrieve CA certificate")
	}

	pp, err := mapParameters(pgExo, pgInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	connDetails, err := connectionDetails(pgExo, ca)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot read connection details")
	}
	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(&pgInstance.Spec.ForProvider, pp, log),
		ConnectionDetails: connDetails,
	}, nil
}

// mapParameters converts a oapi.DbaasServicePg to the internal exoscalev1.PostgreSQLParameters type.
func mapParameters(in oapi.DbaasServicePg, zone exoscalev1.Zone) (*exoscalev1.PostgreSQLParameters, error) {
	settings, err := mapper.ToRawExtension(in.PgSettings)
	if err != nil {
		return nil, fmt.Errorf("unable to parse settings: %w", err)
	}

	return &exoscalev1.PostgreSQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Backup: mapper.ToBackupSpec(in.BackupSchedule),
		Zone:   zone,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: pointer.BoolDeref(in.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: *in.IpFilter,
		},
		Version:    pointer.StringDeref(in.Version, ""),
		PGSettings: settings,
	}, nil
}

func ignoreNotFound(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err.Error() == "resource not found" {
		return nil
	}
	return errors.Wrap(err, "cannot observe instance")
}

func setConditionFromState(pgExo oapi.DbaasServicePg, pgInstance *exoscalev1.PostgreSQL) {
	switch *pgExo.State {
	case oapi.EnumServiceStateRunning:
		pgInstance.SetConditions(exoscalev1.Running())
	case oapi.EnumServiceStateRebuilding:
		pgInstance.SetConditions(exoscalev1.Rebuilding())
	case oapi.EnumServiceStatePoweroff:
		pgInstance.SetConditions(exoscalev1.PoweredOff())
	case oapi.EnumServiceStateRebalancing:
		pgInstance.SetConditions(exoscalev1.Rebalancing())
	}
}

var variantAiven = oapi.EnumPgVariantAiven

// mapObservation fills the status fields from the given response body.
func mapObservation(pg oapi.DbaasServicePg) (exoscalev1.PostgreSQLObservation, error) {
	observation := exoscalev1.PostgreSQLObservation{
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: pointer.BoolDeref(pg.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: pg.Plan,
			},
			IPFilter: *pg.IpFilter,
		},
		Version: pointer.StringDeref(pg.Version, ""),
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: pg.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(pg.Maintenance.Time),
		},
		Backup:     mapper.ToBackupSpec(pg.BackupSchedule),
		NodeStates: mapper.ToNodeStates(pg.NodeStates),
	}

	settings, err := mapper.ToRawExtension(pg.PgSettings)
	if err != nil {
		return observation, errors.Wrap(err, "cannot marshal json")
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
func connectionDetails(pgExo oapi.DbaasServicePg, ca string) (managed.ConnectionDetails, error) {
	raw := pointer.StringDeref(pgExo.Uri, "")
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("cannot parse connection URL: %w", err)
	}
	password, _ := parsed.User.Password()
	return map[string][]byte{
		"POSTGRESQL_USER":     []byte(parsed.User.Username()),
		"POSTGRESQL_PASSWORD": []byte(password),
		"POSTGRESQL_URL":      []byte(raw),
		"POSTGRESQL_DB":       []byte(strings.TrimPrefix(parsed.Path, "/")),
		"POSTGRESQL_HOST":     []byte(parsed.Hostname()),
		"POSTGRESQL_PORT":     []byte(parsed.Port()),
		"ca.crt":              []byte(ca),
	}, nil
}
