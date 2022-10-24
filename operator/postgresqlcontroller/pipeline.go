package postgresqlcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Pipeline provisions IAMKeys on exoscale.com
type Pipeline struct {
	kube     client.Client
	recorder event.Recorder
	exo      *exoscalesdk.Client
}

// NewPipeline returns a new instance of Pipeline.
func NewPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client) *Pipeline {
	return &Pipeline{
		kube:     client,
		recorder: recorder,
		exo:      exoscaleClient,
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.PostgreSQL {
	return mg.(*exoscalev1.PostgreSQL)
}
