package operator

import (
	"github.com/vshn/provider-exoscale/operator/bucketcontroller"
	"github.com/vshn/provider-exoscale/operator/configcontroller"
	"github.com/vshn/provider-exoscale/operator/iamkeycontroller"
	"github.com/vshn/provider-exoscale/operator/mysqlcontroller"
	"github.com/vshn/provider-exoscale/operator/postgresqlcontroller"
	"github.com/vshn/provider-exoscale/operator/rediscontroller"

	ctrl "sigs.k8s.io/controller-runtime"
)

// SetupControllers creates all controllers and adds them to the supplied manager.
func SetupControllers(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		bucketcontroller.SetupController,
		configcontroller.SetupController,
		iamkeycontroller.SetupController,
		mysqlcontroller.SetupController,
		postgresqlcontroller.SetupController,
		rediscontroller.SetupController,
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
		bucketcontroller.SetupWebhook,
		iamkeycontroller.SetupWebhook,
		mysqlcontroller.SetupWebhook,
		postgresqlcontroller.SetupWebhook,
		rediscontroller.SetupWebhook,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
