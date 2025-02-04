package opensearchcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

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

	openSearchInstance := mg.(*exoscalev1.OpenSearch)

	opensearch, err := p.exo.GetDBAASServiceOpensearch(ctx, openSearchInstance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscalesdk.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("cannot observe openSearchInstance: %w", err)
	}

	log.V(1).Info("retrieved openSearchInstance", "state", opensearch.State)

	openSearchInstance.Status.AtProvider, err = mapObservation(opensearch)
	if err != nil {
		log.Error(err, "cannot map openSearchInstance observation, ignoring")
	}
	var state exoscalesdk.EnumServiceState
	if opensearch.State != "" {
		state = opensearch.State
	}
	switch state {
	case exoscalesdk.EnumServiceStateRunning:
		openSearchInstance.SetConditions(exoscalev1.Running())
	case exoscalesdk.EnumServiceStateRebuilding:
		openSearchInstance.SetConditions(exoscalev1.Rebuilding())
	case exoscalesdk.EnumServiceStatePoweroff:
		openSearchInstance.SetConditions(exoscalev1.PoweredOff())
	case exoscalesdk.EnumServiceStateRebalancing:
		openSearchInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown openSearchInstance state", "state", state)
	}

	connDetails, err := connectionDetails(ctx, opensearch, p.exo)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}

	params, err := mapParameters(opensearch, openSearchInstance.Spec.ForProvider.Zone.String())
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse parameters: %w", err)
	}

	currentParams, err := setSettingsDefaults(ctx, *p.exo, &openSearchInstance.Spec.ForProvider)
	if err != nil {
		log.Error(err, "unable to set opensearch settings schema")
		currentParams = &openSearchInstance.Spec.ForProvider
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(currentParams, params, log),
		ConnectionDetails: connDetails,
	}, nil
}

func connectionDetails(ctx context.Context, in *exoscalesdk.DBAASServiceOpensearch, client *exoscalesdk.Client) (managed.ConnectionDetails, error) {
	uriParams := in.URIParams

	password, err := client.RevealDBAASOpensearchUserPassword(ctx, string(in.Name), in.ConnectionInfo.Username)
	if err != nil {
		return nil, fmt.Errorf("cannot reveal password for OpenSearch instance: %w", err)
	}
	return map[string][]byte{
		"OPENSEARCH_USER":          []byte(in.ConnectionInfo.Username),
		"OPENSEARCH_PASSWORD":      []byte(password.Password),
		"OPENSEARCH_HOST":          []byte(uriParams["host"].(string)),
		"OPENSEARCH_PORT":          []byte(uriParams["port"].(string)),
		"OPENSEARCH_URI":           []byte(in.URI),
		"OPENSEARCH_DASHBOARD_URI": []byte(in.ConnectionInfo.DashboardURI),
	}, nil
}

func isUpToDate(current, external *exoscalev1.OpenSearchParameters, log logr.Logger) bool {

	if external == nil {
		return false
	}
	extIPFilter := []string(external.IPFilter)

	checks := map[string]bool{
		"Maintenance":        current.Maintenance.Equals(external.Maintenance),
		"Zone":               current.Zone == external.Zone,
		"Size":               current.Size.Equals(external.Size),
		"IPFilter":           mapper.IsSameStringSet(current.IPFilter, &extIPFilter),
		"OpenSearchSettings": mapper.CompareSettings(current.OpenSearchSettings, external.OpenSearchSettings),
	}
	ok := true
	for k, v := range checks {
		if !v {
			log.V(2).Info("openSearchInstance not up-to-date", "check", k)
			ok = false
		}
	}
	return ok
}

func mapObservation(instance *exoscalesdk.DBAASServiceOpensearch) (exoscalev1.OpenSearchObservation, error) {
	observation := exoscalev1.OpenSearchObservation{
		MajorVersion: instance.Version,
		NodeStates:   mapper.ToNodeStates(&instance.NodeStates),
	}

	jsonSettings, err := json.Marshal(instance.OpensearchSettings)
	if err != nil {
		return exoscalev1.OpenSearchObservation{}, fmt.Errorf("error parsing OpenSearchSettings")
	}

	settings := runtime.RawExtension{Raw: jsonSettings}
	if err != nil {
		return observation, fmt.Errorf("openSearchInstance settings: %w", err)
	}
	observation.OpenSearchSettings = settings

	observation.Maintenance = mapper.ToMaintenance(instance.Maintenance)
	observation.DBaaSParameters = mapper.ToDBaaSParameters(instance.TerminationProtection, instance.Plan, &instance.IPFilter)
	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("openSearchInstance notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in *exoscalesdk.DBAASServiceOpensearch, zone string) (*exoscalev1.OpenSearchParameters, error) {
	jsonSettings, err := json.Marshal(in.OpensearchSettings)

	if err != nil {
		return nil, fmt.Errorf("cannot parse openSearchInstance settings: %w", err)
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	return &exoscalev1.OpenSearchParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Zone: exoscalev1.Zone(zone),
		DBaaSParameters: exoscalev1.DBaaSParameters{
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: in.IPFilter,
		},
		OpenSearchSettings: settings,
	}, nil
}
