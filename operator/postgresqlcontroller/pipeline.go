package postgresqlcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pipeline is a managed.ExternalClient and implements a crossplane reconciler for redis.
type pipeline struct {
	kube     client.Client
	recorder event.Recorder
	exo      *exoscalesdk.Client
}

// newPipeline returns a new instance of pipeline.
func newPipeline(client client.Client, recorder event.Recorder, exoscaleClient *exoscalesdk.Client) *pipeline {
	return &pipeline{
		kube:     client,
		recorder: recorder,
		exo:      exoscaleClient,
	}
}

func fromManaged(mg resource.Managed) *exoscalev1.PostgreSQL {
	return mg.(*exoscalev1.PostgreSQL)
}
