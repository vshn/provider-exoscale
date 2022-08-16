package iamkeycontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/controllerutil"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
	"time"
)

// SetupController adds a controller that reconciles exoscalev1.IAMKey managed resources.
func SetupController(mgr ctrl.Manager) error {
	name := strings.ToLower(exoscalev1.IAMKeyGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(exoscalev1.IAMKeyGroupVersionKind),
		managed.WithExternalConnecter(&IAMKeyConnector{
			controllerutil.GenericConnector{
				Kube:     mgr.GetClient(),
				Recorder: recorder,
			},
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(recorder),
		managed.WithPollInterval(1*time.Hour),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&exoscalev1.IAMKey{}).
		Complete(r)
}

// SetupWebhook adds a webhook for Bucket managed resources.
func SetupWebhook(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&exoscalev1.IAMKey{}).
		WithValidator(&IAMKeyValidator{
			log: mgr.GetLogger().WithName("webhook").WithName(strings.ToLower(exoscalev1.IAMKeyKind)),
		}).
		Complete()
}
