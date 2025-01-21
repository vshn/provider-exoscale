package rediscontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/go-logr/logr"
	controllerruntime "sigs.k8s.io/controller-runtime"
)

func (p pipeline) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	log := controllerruntime.LoggerFrom(ctx)
	log.V(1).Info("observing resource")

	redisInstance := mg.(*exoscalev1.Redis)

	redis, err := p.exo.GetDBAASServiceRedis(ctx, redisInstance.GetInstanceName())
	if err != nil {
		if errors.Is(err, exoscalesdk.ErrNotFound) {
			return managed.ExternalObservation{}, nil
		}
		return managed.ExternalObservation{}, fmt.Errorf("unable to observe instance: %w", err)
	}

	log.V(1).Info("retrieved instance", "state", redis.State)

	redisInstance.Status.AtProvider, err = mapObservation(redis)
	if err != nil {
		log.Error(err, "unable to fully map observation, ignoring.")
	}

	var state exoscalesdk.EnumServiceState
	if redis.State != "" {
		state = redis.State
	}
	switch state {
	case exoscalesdk.EnumServiceStateRunning:
		redisInstance.SetConditions(exoscalev1.Running())
	case exoscalesdk.EnumServiceStateRebuilding:
		redisInstance.SetConditions(exoscalev1.Rebuilding())
	case exoscalesdk.EnumServiceStatePoweroff:
		redisInstance.SetConditions(exoscalev1.PoweredOff())
	case exoscalesdk.EnumServiceStateRebalancing:
		redisInstance.SetConditions(exoscalev1.Rebalancing())
	default:
		log.V(2).Info("ignoring unknown instance state", "state", state)
	}

	rp, err := mapParameters(redis, redisInstance.Spec.ForProvider.Zone)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	cd, err := connectionDetails(ctx, redis, p.exo)
	if err != nil {
		return managed.ExternalObservation{}, fmt.Errorf("unable to parse connection details: %w", err)
	}

	currentParams, err := setSettingsDefaults(ctx, *p.exo, &redisInstance.Spec.ForProvider)
	if err != nil {
		log.Error(err, "unable to set redis settings schema")
		currentParams = &redisInstance.Spec.ForProvider
	}

	observation := managed.ExternalObservation{
		ResourceExists:          true,
		ResourceUpToDate:        isUpToDate(currentParams, rp, log),
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

func connectionDetails(ctx context.Context, in *exoscalesdk.DBAASServiceRedis, client *exoscalesdk.Client) (managed.ConnectionDetails, error) {
	uri := in.URI
	// uri may be absent
	if uri == "" {
		if in.ConnectionInfo == nil || in.ConnectionInfo.URI == nil || len(in.ConnectionInfo.URI) == 0 {
			return map[string][]byte{}, nil
		}
		uri = in.ConnectionInfo.URI[0]
	}
	parsed, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("cannot parse connection URI: %w", err)
	}
	password, err := client.RevealDBAASRedisUserPassword(ctx, string(in.Name), parsed.User.Username())
	if err != nil {
		return nil, fmt.Errorf("cannot reveal password for Redis instance: %w", err)
	}
	return map[string][]byte{
		"REDIS_HOST":     []byte(parsed.Hostname()),
		"REDIS_PORT":     []byte(parsed.Port()),
		"REDIS_USERNAME": []byte(parsed.User.Username()),
		"REDIS_PASSWORD": []byte(password.Password),
		"REDIS_URL":      []byte(uri),
	}, nil
}

func mapObservation(instance *exoscalesdk.DBAASServiceRedis) (exoscalev1.RedisObservation, error) {
	jsonSettings, err := json.Marshal(instance.RedisSettings)
	if err != nil {
		return exoscalev1.RedisObservation{}, fmt.Errorf("error parsing RedisSettings")
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	observation := exoscalev1.RedisObservation{
		Version:    instance.Version,
		NodeStates: mapper.ToNodeStates(&instance.NodeStates),
	}

	observation.RedisSettings = settings

	notifications, err := mapper.ToNotifications(instance.Notifications)
	if err != nil {
		return observation, fmt.Errorf("notifications: %w", err)
	}
	observation.Notifications = notifications

	return observation, nil
}

func mapParameters(in *exoscalesdk.DBAASServiceRedis, zone exoscalev1.Zone) (*exoscalev1.RedisParameters, error) {
	jsonSettings, err := json.Marshal(in.RedisSettings)
	if err != nil {
		return nil, fmt.Errorf("cannot parse redisInstance settings: %w", err)
	}

	settings := runtime.RawExtension{Raw: jsonSettings}

	return &exoscalev1.RedisParameters{
		Maintenance: exoscalev1.MaintenanceSpec{
			DayOfWeek: in.Maintenance.Dow,
			TimeOfDay: exoscalev1.TimeOfDay(in.Maintenance.Time),
		},
		Zone: zone,
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: ptr.Deref(in.TerminationProtection, false),
			Size: exoscalev1.SizeSpec{
				Plan: in.Plan,
			},
			IPFilter: in.IPFilter,
		},
		RedisSettings: settings,
	}, nil
}
