package kafkacontroller

import (
	"strings"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupController adds a controller that reconciles kafka resources.
func SetupController(mgr ctrl.Manager) error {
	name := strings.ToLower(exoscalev1.KafkaGroupKind)
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(exoscalev1.KafkaGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			Kube:     mgr.GetClient(),
			Recorder: recorder,
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

// SetupWebhook adds a webhook for kafka resources.
func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&exoscalev1.Kafka{}).
		WithValidator(&Validator{
			log:  mgr.GetLogger().WithName("webhook").WithName(strings.ToLower(exoscalev1.KafkaKind)),
			kube: mgr.GetClient(),
		}).
		Complete()
}
