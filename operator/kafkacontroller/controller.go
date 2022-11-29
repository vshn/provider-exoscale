package kafkacontroller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type connector struct {
	kube     client.Client
	recorder event.Recorder
}

type connection struct {
	exo oapi.ClientWithResponsesInterface
}

// Connect to the exoscale kafka provider.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("connecting resource")

	kafkaInstance, ok := mg.(*exoscalev1.Kafka)
	if !ok {
		return nil, fmt.Errorf("invalid managed resource type %T for kafka connector", mg)
	}

	exo, err := pipelineutil.OpenExoscaleClient(ctx, c.kube, kafkaInstance.GetProviderConfigName(), exoscalesdk.ClientOptWithAPIEndpoint(fmt.Sprintf("https://api-%s.exoscale.com", kafkaInstance.Spec.ForProvider.Zone)))
	if err != nil {
		return nil, err
	}
	return connection{
		exo: exo.Exoscale,
	}, nil
}

// SetupController adds a controller that reconciles kafka resources.
func SetupController(mgr ctrl.Manager) error {
	name := strings.ToLower(exoscalev1.KafkaGroupKind)
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(exoscalev1.KafkaGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:     mgr.GetClient(),
			recorder: recorder,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(recorder),
		managed.WithPollInterval(1*time.Minute),
		managed.WithConnectionPublishers(cps...),
		managed.WithCreationGracePeriod(30*time.Second))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&exoscalev1.Kafka{}).
		Complete(r)
}
