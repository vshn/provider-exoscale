package kafkacontroller

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/internal/operatortest"
	"github.com/vshn/provider-exoscale/operator/mapper"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestObserve_NotExits(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")

	ctx := context.Background()

	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{Body: []byte{}}, exoscaleapi.ErrNotFound).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.False(t, res.ResourceExists, "report resource not exits")
	})
}

func TestObserve_UpToDate_ConnectionDetails(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	found := sampleAPIKafka("foo")
	found.Uri = pointer.String("foobar.com:21701")
	found.UriParams = &map[string]interface{}{
		"host": "foobar.com",
		"port": "21701",
	}
	found.ConnectionInfo.Nodes = &[]string{
		"10.10.1.1:21701",
		"10.10.1.2:21701",
		"10.10.1.3:21701",
	}
	found.ConnectionInfo.AccessCert = pointer.String("CERT")
	found.ConnectionInfo.AccessKey = pointer.String("KEY")

	ctx := context.Background()

	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.True(t, res.ResourceUpToDate, "report resource uptodate")
		require.NotNil(t, res.ConnectionDetails)
		expectedConnDetails := managed.ConnectionDetails{
			"KAFKA_URI":   []byte("foobar.com:21701"),
			"KAFKA_HOST":  []byte("foobar.com"),
			"KAFKA_PORT":  []byte("21701"),
			"KAFKA_NODES": []byte("10.10.1.1:21701 10.10.1.2:21701 10.10.1.3:21701"),
			"cert.pem":    []byte("CERT"),
			"key.pem":     []byte("KEY"),
		}
		assert.Equal(t, expectedConnDetails, res.ConnectionDetails)
	})
}

func TestObserve_UpToDate_Status(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	found := sampleAPIKafka("foo")
	found.Version = pointer.String("3.2.1")
	found.NodeStates = &[]oapi.DbaasNodeState{
		{
			Name:  "node-1",
			State: "running",
		},
		{
			Name:  "node-3",
			State: "leaving",
		},
	}

	ctx := context.Background()
	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()
	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.True(t, res.ResourceUpToDate, "report resource uptodate")

		assert.Equal(t, "3.2.1", instance.Status.AtProvider.Version)
		require.Len(t, instance.Status.AtProvider.NodeStates, 2, "expect 2 node states")
		assert.Equal(t, "node-1", instance.Status.AtProvider.NodeStates[0].Name)
		assert.EqualValues(t, "running", instance.Status.AtProvider.NodeStates[0].State)
		assert.EqualValues(t, "leaving", instance.Status.AtProvider.NodeStates[1].State)
	})
}

func TestObserve_UpToDate_Condition_NotReady(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	found := sampleAPIKafka("foo")
	state := oapi.EnumServiceStateRebalancing
	found.State = &state

	ctx := context.Background()
	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.True(t, res.ResourceUpToDate, "report resource uptodate")

		readyState := instance.Status.ConditionedStatus.GetCondition(xpv1.TypeReady)

		assert.Equal(t, corev1.ConditionFalse, readyState.Status)
		assert.EqualValues(t, "Rebalancing", readyState.Reason)
	})
}

func TestObserve_UpToDate_Condition_Ready(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	found := sampleAPIKafka("foo")
	state := oapi.EnumServiceStateRunning
	found.State = &state

	ctx := context.Background()
	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.True(t, res.ResourceUpToDate, "report resource uptodate")

		readyState := instance.Status.ConditionedStatus.GetCondition(xpv1.TypeReady)

		assert.Equal(t, corev1.ConditionTrue, readyState.Status)
	})
}

func TestObserve_Outdated(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	found := sampleAPIKafka("foo")
	found.Maintenance.Dow = "tuesday"

	ctx := context.Background()
	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.False(t, res.ResourceUpToDate, "report resource not uptodate")
	})
}

func TestObserve_Outdated_Settings(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	setting, _ := mapper.ToRawExtension(&map[string]interface{}{
    "count": 1,
		"foo": "bar",
	})
	instance.Spec.ForProvider.KafkaSettings = setting
	found := sampleAPIKafka("foo")
	found.KafkaRestSettings = &map[string]interface{}{
		"foo": "bar",
    "count": 2,
	}

	ctx := context.Background()
	exoMock.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName("foo")).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, nil).
		Once()

	assert.NotPanics(t, func() {
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.False(t, res.ResourceUpToDate, "report resource not uptodate")
	})
}

func sampleKafka(name string) exoscalev1.Kafka {
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
	return instance
}

func sampleAPIKafka(name string) *oapi.DbaasServiceKafka {
	res := oapi.DbaasServiceKafka{}

	res.Name = oapi.DbaasServiceName(name)
	res.Plan = "businesss-8"
	res.IpFilter = &[]string{"0.0.0.0/0"}
	res.Maintenance = &oapi.DbaasServiceMaintenance{
		Dow:  "monday",
		Time: "10:10:10",
	}

	nodes := []string{"194.182.160.164:21701",
		"159.100.244.100:21701",
		"159.100.241.65:21701",
	}
	res.ConnectionInfo = &struct {
		AccessCert  *string   "json:\"access-cert,omitempty\""
		AccessKey   *string   "json:\"access-key,omitempty\""
		Nodes       *[]string "json:\"nodes,omitempty\""
		RegistryUri *string   "json:\"registry-uri,omitempty\""
		RestUri     *string   "json:\"rest-uri,omitempty\""
	}{
		AccessCert: pointer.String("SOME ACCESS CERT"),
		AccessKey:  pointer.String("SOME ACCESS KEY"),
		Nodes:      &nodes,
	}

	res.Uri = pointer.String("foo-exoscale-8fa13713-1027-4b9c-bca7-4c14f9ff9928.aivencloud.com")
	res.UriParams = &map[string]interface{}{}

	res.Version = pointer.String("3.2.1")

	return &res
}
