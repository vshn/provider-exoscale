package rediscontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/exoscale/egoscale/v2/oapi"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// pipeline is a managed.ExternalClient and implements a crossplane reconciler for redis.
type pipeline struct {
	kube     client.Client
	recorder event.Recorder
	exo      oapi.ClientWithResponsesInterface
}

// newPipeline returns a new instance of pipeline.
func newPipeline(client client.Client, recorder event.Recorder, exoscaleClient oapi.ClientWithResponsesInterface) *pipeline {
	return &pipeline{
		kube:     client,
		recorder: recorder,
		exo:      exoscaleClient,
	}
}
