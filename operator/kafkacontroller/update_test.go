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
	ctx := context.Background()

	exoMock.On("UpdateDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("bar"),
		mock.MatchedBy(func(req oapi.UpdateDbaasServiceKafkaJSONRequestBody) bool {
			return req.IpFilter != nil && len(*req.IpFilter) == 2 && (*req.IpFilter)[0] == "1.0.0.0/8" &&
				req.Plan != nil && *req.Plan == "businesss-4" &&
				req.Maintenance != nil && req.Maintenance.Dow == "monday" && req.Maintenance.Time == "11:11:11"
		})).
		Return(&oapi.UpdateDbaasServiceKafkaResponse{Body: []byte{}}, nil).
		Once()

	assert.NotPanics(t, func() {
		_, err := c.Update(ctx, &instance)
		require.NoError(t, err)
	})
}

func TestUpdate_invalidInput(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	ctx := context.Background()
	assert.NotPanics(t, func() {
		_, err := c.Update(ctx, nil)
		assert.Error(t, err)
	})
}
