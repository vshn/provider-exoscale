package kafkacontroller

import (
	"context"
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/internal/operatortest"
)

func TestUpdate(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := exoscalev1.Kafka{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bar",
		},
	}
	instance.Spec.ForProvider.Size.Plan = "businesss-4"
	instance.Spec.ForProvider.IPFilter = []string{
		"1.0.0.0/8",
		"2.0.0.0/8",
	}
	instance.Spec.ForProvider.Maintenance.DayOfWeek = "monday"
	instance.Spec.ForProvider.Maintenance.TimeOfDay = "11:11:11"

	updateReq := mockUpdateKafkaCall(exoMock, "bar", nil)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		_, err := c.Update(ctx, &instance)
		require.NoError(t, err)
	})

	if assert.NotNil(t, updateReq.IpFilter) {
		assert.Len(t, *updateReq.IpFilter, 2)
		assert.Equal(t, (*updateReq.IpFilter)[0], "1.0.0.0/8")
	}
	if assert.NotNil(t, updateReq.Plan) {
		assert.Equal(t, *updateReq.Plan, "businesss-4")
	}
	if assert.NotNil(t, updateReq.Maintenance) {
		assert.EqualValues(t, updateReq.Maintenance.Dow, "monday")
		assert.Equal(t, updateReq.Maintenance.Time, "11:11:11")
	}
}

func TestUpdate_invalidInput(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	assert.NotPanics(t, func() {
		ctx := context.Background()
		_, err := c.Update(ctx, nil)
		assert.Error(t, err)
	})
}

func mockUpdateKafkaCall(m *operatortest.ClientWithResponsesInterface, name string, err error) *oapi.UpdateDbaasServiceKafkaJSONRequestBody {
	updateReq := &oapi.UpdateDbaasServiceKafkaJSONRequestBody{}

	m.On("UpdateDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName(name),
		mock.MatchedBy(func(req oapi.UpdateDbaasServiceKafkaJSONRequestBody) bool {
			*updateReq = req
			return true
		})).
		Return(&oapi.UpdateDbaasServiceKafkaResponse{Body: []byte{}}, err).
		Once()

	return updateReq
}
