package postgresqlcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Pipeline provisions IAMKeys on exoscale.com
type Pipeline struct {
	kube           client.Client
	recorder       event.Recorder
	exoscaleClient *exoscalesdk.Client
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client) *Pipeline {
	return &Pipeline{
		kube:           client,
		recorder:       recorder,
		exoscaleClient: exoscaleClient,
	}
}

func toConnectionDetails(instance *oapi.DbaasServicePg) managed.ConnectionDetails {
	return map[string][]byte{
		// TODO: fill in secrets
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.PostgreSQL {
	return mg.(*exoscalev1.PostgreSQL)
}
