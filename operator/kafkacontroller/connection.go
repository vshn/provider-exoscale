package kafkacontroller

import (
	"context"
	"fmt"
	"strings"
	"time"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"github.com/vshn/provider-exoscale/operator/pipelineutil"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscalesdk "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
	controllerruntime "sigs.k8s.io/controller-runtime"
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

// Connect implements managed.ExternalConnecter.
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
		exo:      exo.Exoscale,
	}, nil
}

// SetupController adds a controller that reconciles managed resources.
func SetupController(mgr ctrl.Manager) error {
	name := strings.ToLower(exoscalev1.KafkaGroupKind)
	recorder := event.NewAPIRecorder(mgr.GetEventRecorderFor(name))

	return SetupControllerWithConnecter(mgr, name, recorder, &connector{
		kube:     mgr.GetClient(),
		recorder: recorder,
	}, 30*time.Second)
}

func SetupControllerWithConnecter(mgr ctrl.Manager, name string, recorder event.Recorder, c managed.ExternalConnecter, creationGracePeriod time.Duration) error {
	r := createReconciler(mgr, name, recorder, c, creationGracePeriod)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&exoscalev1.Kafka{}).
		Complete(r)
}

func createReconciler(mgr ctrl.Manager, name string, recorder event.Recorder, c managed.ExternalConnecter, creationGracePeriod time.Duration) *managed.Reconciler {
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	return managed.NewReconciler(mgr,
		resource.ManagedKind(exoscalev1.KafkaGroupVersionKind),
		managed.WithExternalConnecter(c),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithRecorder(recorder),
		managed.WithPollInterval(1*time.Minute),
		managed.WithConnectionPublishers(cps...),
		managed.WithCreationGracePeriod(creationGracePeriod))
}

// Create implements managed.ExternalClient
func (c connection) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("creating resource")

	instance := mg.(*exoscalev1.Kafka)

	spec := instance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.KafkaSettings)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("invalid kafka settings: %w", err)
	}

	body := oapi.CreateDbaasServiceKafkaJSONRequestBody{
		IpFilter:      &ipFilter,
		KafkaSettings: &settings,
		Maintenance: &struct {
			Dow  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow "json:\"dow\""
			Time string                                             "json:\"time\""
		}{
			Dow:  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  spec.Size.Plan,
		TerminationProtection: &spec.TerminationProtection,
	}

	resp, err := c.exo.CreateDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(instance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalCreation{}, fmt.Errorf("unable to create instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return managed.ExternalCreation{}, nil
}

// Delete implements managed.ExternalClient
func (c connection) Delete(ctx context.Context, mg resource.Managed) error {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("deleting resource")

	instance := mg.(*exoscalev1.Kafka)
	resp, err := c.exo.DeleteDbaasServiceWithResponse(ctx, instance.GetInstanceName())
	if err != nil {
		return fmt.Errorf("cannot delete kafak instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return nil
}

// Update implements managed.ExternalClient
func (c connection) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("updating resource")

	instance := mg.(*exoscalev1.Kafka)

	spec := instance.Spec.ForProvider
	ipFilter := []string(spec.IPFilter)
	settings, err := mapper.ToMap(spec.KafkaSettings)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("invalid kafka settings: %w", err)
	}

	body := oapi.UpdateDbaasServiceKafkaJSONRequestBody{
		IpFilter:      &ipFilter,
		KafkaSettings: &settings,
		Maintenance: &struct {
			Dow  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow "json:\"dow\""
			Time string                                             "json:\"time\""
		}{
			Dow:  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow(spec.Maintenance.DayOfWeek),
			Time: spec.Maintenance.TimeOfDay.String(),
		},
		Plan:                  &spec.Size.Plan,
		TerminationProtection: &spec.TerminationProtection,
	}

	resp, err := c.exo.UpdateDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(instance.GetInstanceName()), body)
	if err != nil {
		return managed.ExternalUpdate{}, fmt.Errorf("unable to update instance: %w", err)
	}
	log.V(2).Info("response", "body", string(resp.Body))
	return managed.ExternalUpdate{}, nil
}
