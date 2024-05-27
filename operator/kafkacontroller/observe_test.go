package kafkacontroller

import (
	"context"
	"k8s.io/utils/ptr"
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
	"k8s.io/apimachinery/pkg/runtime"
)

func TestObserve_NotExits(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}

	instance := sampleKafka("foo")
	mockGetKafkaCall(exoMock, "foo", nil, exoscaleapi.ErrNotFound)

	assert.NotPanics(t, func() {
		ctx := context.Background()
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
	found.Uri = ptr.To[string]("foobar.com:21701")
	found.UriParams = &map[string]interface{}{
		"host": "foobar.com",
		"port": "21701",
	}
	found.ConnectionInfo.Nodes = &[]string{
		"10.10.1.1:21701",
		"10.10.1.2:21701",
		"10.10.1.3:21701",
	}
	found.ConnectionInfo.AccessCert = ptr.To[string]("CERT")
	found.ConnectionInfo.AccessKey = ptr.To[string]("KEY")

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)

		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)
		require.NotNil(t, res.ConnectionDetails)
		expectedConnDetails := managed.ConnectionDetails{
			"KAFKA_URI":    []byte("foobar.com:21701"),
			"KAFKA_HOST":   []byte("foobar.com"),
			"KAFKA_PORT":   []byte("21701"),
			"KAFKA_NODES":  []byte("10.10.1.1:21701 10.10.1.2:21701 10.10.1.3:21701"),
			"service.cert": []byte("CERT"),
			"service.key":  []byte("KEY"),
			"ca.crt":       []byte("CA"),
		}
		assert.Equal(t, expectedConnDetails, res.ConnectionDetails)
	})
}

func TestObserve_UpToDate_ConnectionDetails_with_REST(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}

	instance := sampleKafka("foo")
	instance.Spec.ForProvider.KafkaRestEnabled = true
	found := sampleAPIKafka("foo")
	found.Uri = ptr.To[string]("foobar.com:21701")
	found.UriParams = &map[string]interface{}{
		"host": "foobar.com",
		"port": "21701",
	}
	found.ConnectionInfo.Nodes = &[]string{
		"10.10.1.1:21701",
		"10.10.1.2:21701",
		"10.10.1.3:21701",
	}
	found.ConnectionInfo.AccessCert = ptr.To[string]("CERT")
	found.ConnectionInfo.AccessKey = ptr.To[string]("KEY")
	found.KafkaRestEnabled = ptr.To[bool](true)
	found.ConnectionInfo.RestUri = ptr.To[string]("https://admin:BGAUNBS2afjwQ@test.foobar.com:21701")
	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)

		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)
		require.NotNil(t, res.ConnectionDetails)
		expectedConnDetails := managed.ConnectionDetails{
			"KAFKA_URI":      []byte("foobar.com:21701"),
			"KAFKA_REST_URI": []byte("https://admin:BGAUNBS2afjwQ@test.foobar.com:21701"),
			"KAFKA_HOST":     []byte("foobar.com"),
			"KAFKA_PORT":     []byte("21701"),
			"KAFKA_NODES":    []byte("10.10.1.1:21701 10.10.1.2:21701 10.10.1.3:21701"),
			"service.cert":   []byte("CERT"),
			"service.key":    []byte("KEY"),
			"ca.crt":         []byte("CA"),
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
	found.Version = ptr.To[string]("3.2.1")
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

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)

		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)

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

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)

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

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)

		readyState := instance.Status.ConditionedStatus.GetCondition(xpv1.TypeReady)

		assert.Equal(t, corev1.ConditionTrue, readyState.Status)
	})
}

func TestObserve_UpToDate_WithVersion(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	instance.Spec.ForProvider.Version = "3.2"
	found := sampleAPIKafka("foo")
	found.Version = ptr.To[string]("3.2.1")

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)
	})
}

func TestObserve_UpToDate_EmptyRestSettings(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	instance.Spec.ForProvider.KafkaRestEnabled = true
	instance.Spec.ForProvider.KafkaRestSettings = runtime.RawExtension{}
	found := sampleAPIKafka("foo")
	found.KafkaRestEnabled = ptr.To[bool](true)

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)
		assert.Empty(t, res.Diff)
	})
}

func TestObserve_UpToDate_RestSettings(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	restsetting, _ := mapper.ToRawExtension(&map[string]interface{}{
		"producer_acks":                "1",
		"simpleconsumer_pool_size_max": float64(25),
	})
	instance.Spec.ForProvider.KafkaRestEnabled = true
	instance.Spec.ForProvider.KafkaRestSettings = restsetting
	found := sampleAPIKafka("foo")
	found.KafkaRestEnabled = ptr.To[bool](true)

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.Truef(t, res.ResourceUpToDate, "report resource uptodate: Diff: %s", res.Diff)
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

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
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
		"foo":   "bar",
	})
	instance.Spec.ForProvider.KafkaSettings = setting
	found := sampleAPIKafka("foo")
	found.KafkaRestSettings = &map[string]interface{}{
		"foo":   "bar",
		"count": 2,
	}

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
		res, err := c.Observe(ctx, &instance)
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.True(t, res.ResourceExists, "report resource exits")
		assert.False(t, res.ResourceUpToDate, "report resource not uptodate")
	})
}

func TestObserve_Outdated_RestSettings(t *testing.T) {
	exoMock := &operatortest.ClientWithResponsesInterface{}
	c := connection{
		exo: exoMock,
	}
	instance := sampleKafka("foo")
	restsetting, _ := mapper.ToRawExtension(&map[string]interface{}{
		"foo":           "bar",
		"producer_acks": "2",
	})
	instance.Spec.ForProvider.KafkaRestEnabled = true
	instance.Spec.ForProvider.KafkaRestSettings = restsetting
	found := sampleAPIKafka("foo")
	found.KafkaRestEnabled = ptr.To[bool](true)

	mockGetKafkaCall(exoMock, "foo", found, nil)
	mockGetKafkaSettingsCall(exoMock, nil)
	mockCACall(exoMock)

	assert.NotPanics(t, func() {
		ctx := context.Background()
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
	instance.Spec.ForProvider.Version = "3.2"
	instance.Spec.ForProvider.Size.Plan = "businesss-8"
	instance.Spec.ForProvider.IPFilter = []string{
		"0.0.0.0/0",
	}
	instance.Spec.ForProvider.Maintenance.DayOfWeek = "monday"
	instance.Spec.ForProvider.Maintenance.TimeOfDay = "10:10:10"
	instance.Spec.ForProvider.Zone = "ch-dk-2"
	instance.Spec.ForProvider.KafkaSettings = runtime.RawExtension{Raw: []byte(`{"connections_max_idle_ms":60000}`)}
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
	res.KafkaSettings = &map[string]interface{}{
		"connections_max_idle_ms": 60000,
	}
	restSet := map[string]interface{}{}
	for k, v := range defaultRestSettings {
		restSet[k] = v
	}
	res.KafkaRestSettings = &restSet

	nodes := []string{"194.182.160.164:21701",
		"159.100.244.100:21701",
		"159.100.241.65:21701",
	}

	res.ConnectionInfo = &struct {
		AccessCert  *string   "json:\"access-cert,omitempty\""
		AccessKey   *string   "json:\"access-key,omitempty\""
		ConnectUri  *string   "json:\"connect-uri,omitempty\""
		Nodes       *[]string "json:\"nodes,omitempty\""
		RegistryUri *string   "json:\"registry-uri,omitempty\""
		RestUri     *string   "json:\"rest-uri,omitempty\""
	}{
		AccessCert: ptr.To[string]("SOME ACCESS CERT"),
		AccessKey:  ptr.To[string]("SOME ACCESS KEY"),
		Nodes:      &nodes,
	}

	res.Uri = ptr.To[string]("foo-exoscale-8fa13713-1027-4b9c-bca7-4c14f9ff9928.aivencloud.com")
	res.UriParams = &map[string]interface{}{}

	res.Version = ptr.To[string]("3.2.1")

	return &res
}

var defaultRestSettings = map[string]interface{}{
	"consumer_enable_auto_commit":  true,
	"producer_acks":                "1",               // Yes, that's a "1" as a string. I don't know why, that's just how it is..
	"consumer_request_max_bytes":   float64(67108864), // When parsing json into map[string]interface{} we get floats.
	"simpleconsumer_pool_size_max": float64(25),
	"producer_linger_ms":           float64(0),
	"consumer_request_timeout_ms":  float64(1000),
}

func mockGetKafkaCall(m *operatortest.ClientWithResponsesInterface, name string, found *oapi.DbaasServiceKafka, err error) {
	m.On("GetDbaasServiceKafkaWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(&oapi.GetDbaasServiceKafkaResponse{
			Body:    []byte{},
			JSON200: found,
		}, err).
		Once()

}
func mockGetKafkaSettingsCall(m *operatortest.ClientWithResponsesInterface, err error) {
	m.On("GetDbaasSettingsKafkaWithResponse", mock.Anything).
		Return(&oapi.GetDbaasSettingsKafkaResponse{
			Body: rawSettingsResponse,
		}, err).
		Once()
}
func mockCACall(m *operatortest.ClientWithResponsesInterface) {
	m.On("GetDbaasCaCertificateWithResponse", mock.Anything).
		Return(&oapi.GetDbaasCaCertificateResponse{
			JSON200: &struct {
				Certificate *string "json:\"certificate,omitempty\""
			}{
				Certificate: ptr.To[string]("CA"),
			},
		}, nil).
		Once()
}
