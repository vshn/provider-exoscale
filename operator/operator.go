package operator

import (
	"github.com/vshn/provider-exoscale/operator/bucketcontroller"
	"github.com/vshn/provider-exoscale/operator/configcontroller"
	"github.com/vshn/provider-exoscale/operator/iamkeycontroller"
	"github.com/vshn/provider-exoscale/operator/postgresqlcontroller"
	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupControllers creates all controllers and adds them to the supplied manager.
func SetupControllers(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		configcontroller.SetupController,
		iamkeycontroller.SetupController,
		bucketcontroller.SetupController,
		postgresqlcontroller.SetupController,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhooks creates all webhooks and adds them to the supplied manager.
func SetupWebhooks(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		iamkeycontroller.SetupWebhook,
		bucketcontroller.SetupWebhook,
		postgresqlcontroller.SetupWebhook,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
