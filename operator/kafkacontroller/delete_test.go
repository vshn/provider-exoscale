// go:build ignore

package kafkacontroller

import (
	"context"
	"testing"

	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/internal/operatortest"
)

func TestDelete(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := exoscalev1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buzz",
		},
	}

	mockDeleteKafkaCall(exoMock, "buzz", nil)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		err := c.Delete(ctx, &instance)
		require.NoError(t, err)
	})
}
func TestDelete_invalidInput(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	assert.NotPanics(t, func() {
		ctx := context.Background()
		err := c.Delete(ctx, nil)
		assert.Error(t, err)
	})
}
func TestDelete_Idempotent(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := exoscalev1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "buzz",
		},
	}

	mockDeleteKafkaCall(exoMock, "buzz", exoscaleapi.ErrNotFound)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		err := c.Delete(ctx, &instance)
		require.NoError(t, err)
	})
}

func mockDeleteKafkaCall(m *operatortest.ClientWithResponsesInterface, name string, err error) {
	m.On("DeleteDbaasServiceWithResponse", mock.Anything, name).
		Return(&oapi.DeleteDbaasServiceResponse{Body: []byte{}}, err).
		Once()
}
