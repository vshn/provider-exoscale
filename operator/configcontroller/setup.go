package configcontroller

import (
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	providerv1 "github.com/vshn/provider-exoscale/apis/provider/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/providerconfig"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// +kubebuilder:rbac:groups=exoscale.crossplane.io,resources=providerconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=exoscale.crossplane.io,resources=providerconfigs/status;providerconfigs/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=exoscale.crossplane.io,resources=providerconfigusages,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=exoscale.crossplane.io,resources=providerconfigusages/status;providerconfigusages/finalizers,verbs=get;update;patch

// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;create;update
// +kubebuilder:rbac:groups=core,resources=events,verbs=create

// SetupController adds a controller that reconciles ProviderConfigs by accounting for their current usage.
func SetupController(mgr ctrl.Manager) error {
	name := providerconfig.ControllerName(providerv1.ProviderConfigGroupKind)

	of := resource.ProviderConfigKinds{
		Config:    providerv1.ProviderConfigGroupVersionKind,
		UsageList: providerv1.ProviderConfigUsageListGroupVersionKind,
	}

	r := providerconfig.NewReconciler(mgr, of,
		providerconfig.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		providerconfig.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&providerv1.ProviderConfig{}).
		Watches(&source.Kind{Type: &providerv1.ProviderConfigUsage{}}, &resource.EnqueueRequestForProviderConfig{}).
		Complete(r)
}
