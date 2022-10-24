package bucketcontroller

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

// SetupController adds a controller that reconciles exoscalev1.Bucket managed resources.
func SetupController(mgr ctrl.Manager) error {
	name := managed.ControllerName(exoscalev1.BucketGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(exoscalev1.BucketGroupVersionKind),
		managed.WithExternalConnecter(&bucketConnector{
			Kube:     mgr.GetClient(),
			Recorder: recorder,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(recorder),
		managed.WithPollInterval(1*time.Hour), // buckets are rather static
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&exoscalev1.Bucket{}).
		Complete(r)
}

// SetupWebhook adds a webhook for Bucket managed resources.
func SetupWebhook(mgr ctrl.Manager) error {
	/*
		Totally undocumented and hard-to-find feature is that the builder automatically registers the URL path for the webhook.
		What's more, not even the tests in upstream controller-runtime reveal what this path is _actually_ going to look like.
		So here's how the path is built (dots replaced with dash, lower-cased, single-form):
		 /validate-<group>-<version>-<kind>
		 /mutate-<group>-<version>-<kind>
		Example:
		 /validate-exoscale-crossplane-io-v1-bucket
		This path has to be given in the `//+kubebuilder:webhook:...` magic comment, see example:
		 +kubebuilder:webhook:verbs=create;update;delete,path=/validate-exoscale-crossplane-io-v1-bucket,mutating=false,failurePolicy=fail,groups=exoscale.crossplane.io,resources=buckets,versions=v1alpha1,name=buckets.exoscale.crossplane.io,sideEffects=None,admissionReviewVersions=v1
		Pay special attention to the plural forms and correct versions!
	*/
	return ctrl.NewWebhookManagedBy(mgr).
		For(&exoscalev1.Bucket{}).
		WithValidator(&BucketValidator{
			log: mgr.GetLogger().WithName("webhook").WithName(strings.ToLower(exoscalev1.BucketKind)),
		}).
		Complete()
}
