package opensearchcontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	controllerruntime "sigs.k8s.io/controller-runtime"
)

// Create implements managed.ExternalClient.
func (p *pipeline) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.Info("creating resource")

	openSearch := mg.(*exoscalev1.OpenSearch)
	forProvider := openSearch.Spec.ForProvider
	ipFilter := []string(forProvider.IPFilter)
	settings := exoscalesdk.JSONSchemaOpensearch{}
	if len(forProvider.OpenSearchSettings.Raw) != 0 {
		err := json.Unmarshal(forProvider.OpenSearchSettings.Raw, &settings)
		if err != nil {
			return managed.ExternalCreation{}, fmt.Errorf("cannot map opensearchInstance settings: %w", err)
		}
	}

	body := exoscalesdk.CreateDBAASServiceOpensearchRequest{
		Plan:     forProvider.Size.Plan,
		IPFilter: ipFilter,
		Maintenance: &exoscalesdk.CreateDBAASServiceOpensearchRequestMaintenance{
			Dow:  exoscalesdk.CreateDBAASServiceOpensearchRequestMaintenanceDow(forProvider.Maintenance.DayOfWeek),
			Time: forProvider.Maintenance.TimeOfDay.String(),
		},
		OpensearchSettings:    settings,
		TerminationProtection: &forProvider.TerminationProtection,
		// majorVersion can be only major: ['1','2']
		Version: forProvider.MajorVersion,
	}
	resp, err := p.exo.CreateDBAASServiceOpensearch(ctx, openSearch.GetInstanceName(), body)
	if err != nil {
		if strings.Contains(err.Error(), "Service name is already taken") {
			// According to the ExternalClient Interface, create needs to be idempotent.
			// However the exoscale client doesn't return very helpful errors, so we need to make this brittle matching to find if we get an already exits error
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, fmt.Errorf("cannot create OpenSearch Instance: %v, \nerr: %w", openSearch.GetInstanceName(), err)
	}
	log.V(1).Info("resource created", "message", resp.Message)
	return managed.ExternalCreation{}, nil
}
