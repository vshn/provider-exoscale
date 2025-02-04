package opensearchcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	exoscalesdk "github.com/exoscale/egoscale/v3"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pipeline is a managed.ExternalClient and implements a crossplane reconciler for MySQL.
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
