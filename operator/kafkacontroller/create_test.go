package kafkacontroller

import (
	"context"
	"fmt"
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

func TestCreate(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := exoscalev1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	instance.Spec.ForProvider.Size.Plan = "businesss-8"
	instance.Spec.ForProvider.IPFilter = []string{
		"0.0.0.0/0",
	}
	instance.Spec.ForProvider.Maintenance.DayOfWeek = "monday"
	instance.Spec.ForProvider.Maintenance.TimeOfDay = "10:10:10"
	ctx := context.Background()

	createReq := mockCreateKafkaCall(exoMock, "foo", nil)

	assert.NotPanics(t, func() {
		_, err := c.Create(ctx, &instance)
		require.NoError(t, err)
	})
	if assert.NotNil(t, createReq.IpFilter) {
		assert.Len(t, *createReq.IpFilter, 1)
		assert.Equal(t, (*createReq.IpFilter)[0], "0.0.0.0/0")
	}
	if assert.NotNil(t, createReq.Plan) {
		assert.Equal(t, createReq.Plan, "businesss-8")
	}
	if assert.NotNil(t, createReq.Maintenance) {
		assert.EqualValues(t, createReq.Maintenance.Dow, "monday")
		assert.Equal(t, createReq.Maintenance.Time, "10:10:10")
	}
}

func TestCreate_Idempotent(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := exoscalev1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	instance.Spec.ForProvider.Size.Plan = "businesss-8"
	instance.Spec.ForProvider.IPFilter = []string{
		"0.0.0.0/0",
	}
	instance.Spec.ForProvider.Maintenance.DayOfWeek = "monday"
	instance.Spec.ForProvider.Maintenance.TimeOfDay = "10:10:10"

	_ = mockCreateKafkaCall(exoMock, "foo", fmt.Errorf("%w: Service name is already taken.", exoscaleapi.ErrInvalidRequest))

	assert.NotPanics(t, func() {
		ctx := context.Background()
		_, err := c.Create(ctx, &instance)
		require.NoError(t, err)
	})
}

func TestCreate_invalidInput(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	ctx := context.Background()
	assert.NotPanics(t, func() {
		_, err := c.Create(ctx, nil)
		assert.Error(t, err)
	})
}

func mockCreateKafkaCall(m *operatortest.ClientWithResponsesInterface, name string, err error) *oapi.CreateDbaasServiceKafkaJSONRequestBody {
	createReq := &oapi.CreateDbaasServiceKafkaJSONRequestBody{}

	m.On("CreateDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName(name),
		mock.MatchedBy(func(req oapi.CreateDbaasServiceKafkaJSONRequestBody) bool {
			*createReq = req
			return true
		})).
		Return(&oapi.CreateDbaasServiceKafkaResponse{Body: []byte{}}, err).
		Once()

	return createReq
}
