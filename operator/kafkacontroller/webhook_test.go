// go:build ignore

package kafkacontroller

import (
	"context"
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/mock"
	"github.com/vshn/provider-exoscale/internal/operatortest"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
)

func TestWebhook_Create(t *testing.T) {
	ctx := context.TODO()
	v := Validator{
		log: logr.Discard(),
	}

	exoMock := &operatortest.ClientWithResponsesInterface{}
	mockAvailableVersionsCall(exoMock)
	base := sampleKafka("foo")

	t.Run("valid", func(t *testing.T) {
		err := v.validateCreateWithExoClient(ctx, &base, exoMock)
		assert.NoError(t, err)
	})
	t.Run("invalid empty", func(t *testing.T) {
		err := v.validateCreateWithExoClient(ctx, &exoscalev1.Kafka{}, exoMock)
		assert.Error(t, err)
	})
	t.Run("invalid no ipfilter", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.IPFilter = nil
		err := v.validateCreateWithExoClient(ctx, &inst, exoMock)
		assert.Error(t, err)
	})
	t.Run("invalid no time", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.Maintenance.TimeOfDay = ""
		err := v.validateCreateWithExoClient(ctx, &inst, exoMock)
		assert.Error(t, err)
	})
	t.Run("invalid version", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.Version = "non-valid-version"
		err := v.validateCreateWithExoClient(ctx, &inst, exoMock)
		assert.Error(t, err)
	})
}

func TestWebhook_Update(t *testing.T) {
	ctx := context.TODO()
	v := Validator{
		log: logr.Discard(),
	}

	base := sampleKafka("foo")

	t.Run("no change", func(t *testing.T) {
		_, err := v.ValidateUpdate(ctx, &base, &base)
		assert.NoError(t, err)
	})
	t.Run("valid change", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.IPFilter = []string{"10.10.1.1/24", "10.10.2.1/24"}
		_, err := v.ValidateUpdate(ctx, &base, &inst)
		assert.NoError(t, err)
	})
	t.Run("remove ipfilter", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.IPFilter = nil
		_, err := v.ValidateUpdate(ctx, &base, &inst)
		assert.Error(t, err)
	})
	t.Run("change zone", func(t *testing.T) {
		inst := base
		inst.Spec.ForProvider.Zone = "ch-gva-2"
		_, err := v.ValidateUpdate(ctx, &base, &inst)
		assert.Error(t, err)
	})
	t.Run("change unsupported version", func(t *testing.T) {
		newInst := base
		oldInst := base

		oldInst.Status.AtProvider.Version = "3.2.1"
		newInst.Spec.ForProvider.Version = "3.3"

		_, err := v.ValidateUpdate(ctx, &oldInst, &newInst)
		assert.Error(t, err)
	})
	t.Run("change supported version", func(t *testing.T) {
		newInst := base
		oldInst := base

		oldInst.Status.AtProvider.Version = "3.2.1"
		newInst.Spec.ForProvider.Version = "3.2"

		_, err := v.ValidateUpdate(ctx, &oldInst, &newInst)
		assert.NoError(t, err)
	})

}

func mockAvailableVersionsCall(m *operatortest.ClientWithResponsesInterface) {
	m.On("GetDbaasServiceTypeWithResponse", mock.Anything, mock.Anything).
		Return(&oapi.GetDbaasServiceTypeResponse{
			JSON200: &oapi.DbaasServiceType{
				AvailableVersions: &[]string{"3.2", "4.3"},
			},
		}, nil)
}
