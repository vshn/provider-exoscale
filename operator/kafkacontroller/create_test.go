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

	exoMock.On("CreateDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo"),
		mock.MatchedBy(func(req oapi.CreateDbaasServiceKafkaJSONRequestBody) bool {
			return req.IpFilter != nil && len(*req.IpFilter) == 1 && (*req.IpFilter)[0] == "0.0.0.0/0" &&
				req.Plan == "businesss-8" &&
				req.Maintenance != nil && req.Maintenance.Dow == "monday" && req.Maintenance.Time == "10:10:10"
		})).
		Return(&oapi.CreateDbaasServiceKafkaResponse{Body: []byte{}}, nil).
		Once()

	assert.NotPanics(t, func() {
		_, err := c.Create(ctx, &instance)
		require.NoError(t, err)
	})
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
	ctx := context.Background()

	exoMock.On("CreateDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo"), mock.Anything).
		Return(nil, fmt.Errorf("%w: Service name is already taken.", exoscaleapi.ErrInvalidRequest)).
		Once()

	assert.NotPanics(t, func() {
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
