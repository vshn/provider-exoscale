package opensearchcontroller

import (
	"context"
	"errors"
	"fmt"
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

	openSearchInstance := mg.(*exoscalev1.OpenSearch)

	resp, err := p.exo.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(openSearchInstance.GetInstanceName()))
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrNotFound) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("cannot observe openSearchInstance: %w", err)
	}

	opensearch := *resp.JSON200
	log.V(2).Info("response", "raw", string(resp.Body))
	log.V(1).Info("retrieved openSearchInstance", "state", opensearch.State)

	openSearchInstance.Status.AtProvider, err = mapObservation(opensearch)
	if err != nil {
		log.Error(err, "cannot map openSearchInstance observation, ignoring")
	}
	var state oapi.EnumServiceState
	if opensearch.State != nil {
		state = *opensearch.State
	}
	switch state {
	case oapi.EnumServiceStateRunning:
		openSearchInstance.SetConditions(exoscalev1.Running())
	case oapi.EnumServiceStateRebuilding:
		openSearchInstance.SetConditions(exoscalev1.Rebuilding())
	case oapi.EnumServiceStatePoweroff:
		openSearchInstance.SetConditions(exoscalev1.PoweredOff())
	case oapi.EnumServiceStateRebalancing:
		openSearchInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown openSearchInstance state", "state", state)
	}

	connDetails, err := connectionDetails(opensearch)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse connection details: %w", err)
	}

	params, err := mapParameters(opensearch, openSearchInstance.Spec.ForProvider.Zone.String())
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("cannot parse parameters: %w", err)
	}

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  isUpToDate(&openSearchInstance.Spec.ForProvider, params, log),
		ConnectionDetails: connDetails,
	}, nil
}

func connectionDetails(in oapi.DbaasServiceOpensearch) (managed.ConnectionDetails, error) {
	return map[string][]byte{
		"OPENSEARCH_USER":          []byte(*in.ConnectionInfo.Username),
		"OPENSEARCH_PASSWORD":      []byte(*in.ConnectionInfo.Password),
		"OPENSEARCH_URI":           []byte(*in.Uri),
		"OPENSEARCH_DASHBOARD_URI": []byte(*in.ConnectionInfo.DashboardUri),
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

func mapObservation(instance oapi.DbaasServiceOpensearch) (exoscalev1.OpenSearchObservation, error) {
	observation := exoscalev1.OpenSearchObservation{
		Version:    pointer.StringDeref(instance.Version, ""),
		NodeStates: mapper.ToNodeStates(instance.NodeStates),
	}

	settings, err := mapper.ToRawExtension(instance.OpensearchSettings)
	if err != nil {
		return observation, fmt.Errorf("openSearchInstance settings: %w", err)
	}
	observation.OpenSearchSettings = settings

	observation.Maintenance = mapper.ToMaintenance(instance.Maintenance)
	observation.DBaaSParameters = mapper.ToDBaaSParameters(instance.TerminationProtection, instance.Plan, instance.IpFilter)
	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("openSearchInstance notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in oapi.DbaasServiceOpensearch, zone string) (*exoscalev1.OpenSearchParameters, error) {
	settings, err := mapper.ToRawExtension(in.OpensearchSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot parse openSearchInstance settings: %w", err)
	}
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
			IPFilter: mapper.ToSlice(in.IpFilter),
		},
		OpenSearchSettings: settings,
	}, nil
}
