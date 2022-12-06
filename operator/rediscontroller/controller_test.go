//go:build integration

package rediscontroller

import (
	"fmt"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/internal/operatortest"
	"github.com/vshn/provider-exoscale/operator/mapper"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type RedisControllerTestSuite struct {
	operatortest.Suite

	// wait time between reconciles
	reconcileWaitPeriod time.Duration

	p          *pipeline
	reconciler *managed.Reconciler
}

func (ts *RedisControllerTestSuite) SetupTest() {
	ts.reconcileWaitPeriod = 10 * time.Millisecond

	nopRecorder := event.NewNopRecorder()
	ts.p = newPipeline(ts.Client, nopRecorder, ts.ExoClientMock)

	ts.reconciler = createReconciler(ts.Manager, "testredis", nopRecorder, &connector{
		kube:     ts.Client,
		recorder: nopRecorder,
		p:        ts.p,
	}, 0)
}

func (ts *RedisControllerTestSuite) TestCreate() {
	name := "test-create"

	settings := map[string]any{
		"maxmemory_policy": "noeviction",
	}

	mg := newRedisInstance(name, settings)

	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(nil, exoscaleapi.ErrNotFound).
		Once()
	ts.ExoClientMock.On("CreateDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name), mock.Anything).
		Return(&oapi.CreateDbaasServiceRedisResponse{Body: []byte{}}, nil).
		Once()

	redisResponse := ts.getRedisResponse(mg, settings)
	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(redisResponse, nil)

	ts.EnsureResources(mg)

	// first request creation
	_, err := ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	// second reconcile should've setup everything correctly.
	_, err = ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	secret := &corev1.Secret{}
	ts.FetchResource(types.NamespacedName{Name: mg.Spec.WriteConnectionSecretToReference.Name, Namespace: mg.Spec.WriteConnectionSecretToReference.Namespace}, secret)

	ts.Assert().Equal("https://foo:bar@baz:5321", string(secret.Data["REDIS_URL"]))
	ts.Assert().Equal("foo", string(secret.Data["REDIS_USERNAME"]))
	ts.Assert().Equal("bar", string(secret.Data["REDIS_PASSWORD"]))
	ts.Assert().Equal("baz", string(secret.Data["REDIS_HOST"]))
	ts.Assert().Equal("5321", string(secret.Data["REDIS_PORT"]))

	instance := &exoscalev1.Redis{}
	ts.FetchResource(types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}, instance)

	ready := false
	synced := false
	for _, condition := range instance.Status.Conditions {
		if condition.Type == xpv1.TypeReady {
			ts.Assert().Equal(corev1.ConditionTrue, condition.Status)
			ts.Assert().Equal(xpv1.ReasonAvailable, condition.Reason)
			ready = true
		} else if condition.Type == xpv1.TypeSynced {
			ts.Assert().Equal(corev1.ConditionTrue, condition.Status)
			ts.Assert().Equal(xpv1.ReasonReconcileSuccess, condition.Reason)
			synced = true
		}
	}
	ts.Assert().True(ready && synced)

	ts.ExoClientMock.AssertExpectations(ts.T())
}

func (ts *RedisControllerTestSuite) TestUpdate() {
	name := "test-update"
	settings := map[string]any{
		"maxmemory_policy": "noeviction",
	}

	mg := newRedisInstance(name, settings)
	mg.Status = exoscalev1.RedisStatus{
		ResourceStatus: xpv1.ResourceStatus{},
		AtProvider:     exoscalev1.RedisObservation{},
	}
	redisResponse := ts.getRedisResponse(mg, settings)

	ts.ExoClientMock.On("UpdateDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name), mock.Anything).
		Return(&oapi.UpdateDbaasServiceRedisResponse{Body: []byte{}}, nil).
		Once()

	// once the "old" response and then the updated response.
	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(redisResponse, nil).Once()

	ts.EnsureResources(mg)

	updatedResponse := ts.getRedisResponse(mg, settings)
	updatedResponse.JSON200.Maintenance.Dow = oapi.DbaasServiceMaintenanceDowFriday
	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(updatedResponse, nil)

	instance := &exoscalev1.Redis{}
	ts.FetchResource(types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}, instance)

	instance.Spec.ForProvider.Maintenance.DayOfWeek = oapi.DbaasServiceMaintenanceDowFriday

	ts.UpdateResources(instance)

	// requested update
	_, err := ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	// second reconcile should be updated
	_, err = ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	instance = &exoscalev1.Redis{}
	ts.FetchResource(types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}, instance)

	ts.Assert().Equal(instance.Spec.ForProvider.Maintenance.DayOfWeek, oapi.DbaasServiceMaintenanceDowFriday)

	ready := false
	synced := false
	for _, condition := range instance.Status.Conditions {
		if condition.Type == xpv1.TypeReady {
			ts.Assert().Equal(corev1.ConditionTrue, condition.Status)
			ts.Assert().Equal(xpv1.ReasonAvailable, condition.Reason)
			ready = true
		} else if condition.Type == xpv1.TypeSynced {
			ts.Assert().Equal(corev1.ConditionTrue, condition.Status)
			ts.Assert().Equal(xpv1.ReasonReconcileSuccess, condition.Reason)
			synced = true
		}
	}
	ts.Assert().True(ready && synced, "not ready & synced (ready: %t / synced: %t)", ready, synced)

	ts.ExoClientMock.AssertExpectations(ts.T())
}

func (ts *RedisControllerTestSuite) TestDelete() {
	name := "test-delete"
	settings := map[string]any{
		"maxmemory_policy": "noeviction",
	}

	mg := newRedisInstance(name, settings)
	mg.Status = exoscalev1.RedisStatus{
		ResourceStatus: xpv1.ResourceStatus{},
		AtProvider:     exoscalev1.RedisObservation{},
	}
	redisResponse := ts.getRedisResponse(mg, settings)

	ts.EnsureResources(mg)

	// reply once with a still existing redis, to allow deletion to happen.
	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(redisResponse, nil).Twice()
	ts.ExoClientMock.On("GetDbaasServiceRedisWithResponse", mock.Anything, oapi.DbaasServiceName(name)).
		Return(nil, exoscaleapi.ErrNotFound)

	// external resource is up-to-date
	_, err := ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	ts.ExoClientMock.On("DeleteDbaasServiceWithResponse", mock.Anything, name).
		Return(&oapi.DeleteDbaasServiceResponse{Body: []byte{}}, nil).
		Once()

	ts.DeleteResources(mg)

	// requested deletion of external resource
	_, err = ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	// removes finalizer
	_, err = ts.reconcile(reconcile.Request{NamespacedName: types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}})
	ts.Assert().NoError(err)

	instance := &exoscalev1.Redis{}
	err = ts.Client.Get(ts.Context, types.NamespacedName{Name: mg.Name, Namespace: mg.Namespace}, instance)
	ts.Assert().True(apierrors.IsNotFound(err))

	ts.ExoClientMock.AssertExpectations(ts.T())
}

// reconcile waits reconcileWaitPeriod before running a reconciliation.
func (ts *RedisControllerTestSuite) reconcile(req reconcile.Request) (reconcile.Result, error) {
	ts.T().Helper()
	// wait a bit so that status updates are done in the background (by the manager?)
	time.Sleep(ts.reconcileWaitPeriod)
	return ts.reconciler.Reconcile(ts.Context, req)
}

// resetExoClientMock is a hacky workaround to get rid of calls without `.Times()` specifier (or `.Once()` / `.Twice()`).
// Calls without times specifier are necessary to have since the reconciliation can happen many times between the calls.
// Specifying the amount of times `Observe()` calls `Get` requests would be brittle since that may change depending on factors
// we don't know.
// .Unset() exists for this case but has a bug: https://github.com/stretchr/testify/issues/1236
// Therefore we just reset the ExpectedCalls (without mutex since that is private) and re-add the mocked calls
// which are always necessary to have.
func (ts *RedisControllerTestSuite) resetExoClientMock() {
	ts.ExoClientMock.ExpectedCalls = nil
}

func (ts *RedisControllerTestSuite) getRedisResponse(mg *exoscalev1.Redis, settings map[string]any) *oapi.GetDbaasServiceRedisResponse {
	providerSpec := mg.Spec.ForProvider
	master := oapi.DbaasNodeStateRoleMaster
	running := oapi.EnumServiceStateRunning

	return &oapi.GetDbaasServiceRedisResponse{
		JSON200: &oapi.DbaasServiceRedis{
			Maintenance: &oapi.DbaasServiceMaintenance{
				Dow:  providerSpec.Maintenance.DayOfWeek,
				Time: providerSpec.Maintenance.TimeOfDay.String(),
			},
			Name: oapi.DbaasServiceName(mg.Name),
			NodeStates: &[]oapi.DbaasNodeState{
				{
					Name:  "test-1",
					Role:  &master,
					State: oapi.DbaasNodeStateStateRunning,
				},
			},
			IpFilter:              &[]string{providerSpec.IPFilter[0]},
			Plan:                  providerSpec.Size.Plan,
			RedisSettings:         &settings,
			State:                 &running,
			TerminationProtection: pointer.Bool(false),
			ConnectionInfo: &struct {
				Password *string   `json:"password,omitempty"`
				Slave    *[]string `json:"slave,omitempty"`
				Uri      *[]string `json:"uri,omitempty"`
			}{
				Uri: &[]string{"https://foo:bar@baz:5321"},
			},
			Uri:       nil,
			UriParams: nil,
			Users:     nil,
			Version:   nil,
		},
	}
}

func newRedisInstance(name string, settings map[string]any) *exoscalev1.Redis {
	rs, err := mapper.ToRawExtension(&settings)
	if err != nil {
		panic(fmt.Errorf("settings not convertible to raw extension: %w", err))
	}

	return &exoscalev1.Redis{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: exoscalev1.RedisSpec{
			ResourceSpec: xpv1.ResourceSpec{
				WriteConnectionSecretToReference: &xpv1.SecretReference{
					Name:      name,
					Namespace: "default",
				},
			},
			ForProvider: exoscalev1.RedisParameters{
				Maintenance: exoscalev1.MaintenanceSpec{
					DayOfWeek: oapi.DbaasServiceMaintenanceDowMonday,
					TimeOfDay: "12:00:00",
				},
				Zone: "ch-dk-2",
				DBaaSParameters: exoscalev1.DBaaSParameters{
					TerminationProtection: false,
					Size: exoscalev1.SizeSpec{
						Plan: "testplan",
					},
					IPFilter: exoscalev1.IPFilter{"0.0.0.0/0"},
				},
				RedisSettings: rs,
			},
		},
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestRedisControllerTestSuite(t *testing.T) {
	suite.Run(t, new(RedisControllerTestSuite))
}
