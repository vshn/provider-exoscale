package rediscontroller

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	exoscaleapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/go-logr/logr"
	"k8s.io/utils/pointer"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (p pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	redisInstance := mg.(*exoscalev1.Redis)

	resp, err := p.exo.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(redisInstance.GetInstanceName()))
	if err != nil {
		if errors.Is(err, exoscaleapi.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("unable to observe instance: %w", err)
	}
	redis := *resp.JSON200

	log.V(2).Info("response", "raw", string(resp.Body))
	log.V(1).Info("retrieved instance", "state", redis.State)

	redisInstance.Status.AtProvider, err = mapObservation(redis)
	if err != nil {
		log.Error(err, "unable to fully map observation, ignoring.")
	}

	var state oapi.EnumServiceState
	if redis.State != nil {
		state = *redis.State
	}
	switch state {
	case oapi.EnumServiceStateRunning:
		redisInstance.SetConditions(exoscalev1.Running())
	case oapi.EnumServiceStateRebuilding:
		redisInstance.SetConditions(exoscalev1.Rebuilding())
	case oapi.EnumServiceStatePoweroff:
		redisInstance.SetConditions(exoscalev1.PoweredOff())
	case oapi.EnumServiceStateRebalancing:
		redisInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown instance state", "state", state)
	}

	res, err := p.exo.GetDbaasCaCertificateWithResponse(ctx)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("unable to retrieve CA certificate: %w", err)
	}
	ca := *res.JSON200.Certificate

	rp, err := mapParameters(redis, redisInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	cd, err := connectionDetails(redis, ca)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("unable to parse connection details: %w", err)
	}

	observation := managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(&redisInstance.Spec.ForProvider, rp, log),
		ResourceLateInitialized: false,
		ConnectionDetails:       cd,
	}

	return observation, nil
}

func isUpToDate(current, external *exoscalev1.RedisParameters, log logr.Logger) bool {
	if external == nil {
		return false
	}
	extIPFilter := []string(external.IPFilter)
	checks := map[string]bool{
		"IPFilter":              mapper.IsSameStringSet(current.IPFilter, &extIPFilter),
		"Maintenance":           current.Maintenance.Equals(external.Maintenance),
		"Size":                  current.Size.Equals(external.Size),
		"TerminationProtection": current.TerminationProtection == external.TerminationProtection,
		"RedisSettings":         mapper.CompareSettings(current.RedisSettings, external.RedisSettings),
	}
	ok := true
	for _, v := range checks {
		if !v {
			log.V(2).Info("instance not up-to-date", "check", v)
			ok = false
		}
	}
	return ok
}

func connectionDetails(in oapi.DbaasServiceRedis, ca string) (map[string][]byte, error) {
	if in.ConnectionInfo == nil || len(*in.ConnectionInfo.Uri) == 0 {
		return map[string][]byte{}, nil
	}
	uri := (*in.ConnectionInfo.Uri)[0]
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	password, _ := parsed.User.Password()
	return map[string][]byte{
		"REDIS_HOST":     []byte(parsed.Hostname()),
		"REDIS_PORT":     []byte(parsed.Port()),
		"REDIS_USERNAME": []byte(parsed.User.Username()),
		"REDIS_PASSWORD": []byte(password),
		"REDIS_URL":      []byte(uri),
		"ca.crt":         []byte(ca),
	}, nil
}

func mapObservation(instance oapi.DbaasServiceRedis) (exoscalev1.RedisObservation, error) {
	observation := exoscalev1.RedisObservation{
		Version:    pointer.StringDeref(instance.Version, ""),
		NodeStates: mapper.ToNodeStates(instance.NodeStates),
	}

	settings, err := mapper.ToRawExtension(instance.RedisSettings)
	if err != nil {
		return observation, fmt.Errorf("settings: %w", err)
	}
	observation.RedisSettings = settings

	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in oapi.DbaasServiceRedis, zone exoscalev1.Zone) (*exoscalev1.RedisParameters, error) {
	settings, err := mapper.ToRawExtension(in.RedisSettings)
	if err != nil {
		return nil, fmt.Errorf("unable to parse settings: %w", err)
	}
	return &exoscalev1.RedisParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Zone: zone,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: pointer.BoolDeref(in.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: *in.IpFilter,
		},
		RedisSettings: settings,
	}, nil
}
