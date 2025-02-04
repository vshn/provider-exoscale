package rediscontroller

import (
	"context"
	"encoding/json"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDBAASSettingsRedis(ctx context.Context) (*exoscalesdk.GetDBAASSettingsRedisResponse, error)
}

func setSettingsDefaults(ctx context.Context, f settingsFetcher, in *exoscalev1.RedisParameters) (*exoscalev1.RedisParameters, error) {
	s, err := fetchSettingSchema(ctx, f)
	if err != nil {
		return nil, err
	}
	res := in.DeepCopy()

	res.RedisSettings, err = s.SetDefaults("redis", res.RedisSettings)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func fetchSettingSchema(ctx context.Context, f settingsFetcher) (settings.Schemas, error) {
	resp, err := f.GetDBAASSettingsRedis(ctx)
	if err != nil {
		return nil, err
	}
	settingsJson, err := json.Marshal(resp)
	if err != nil {
		return nil, err
	}
	schemas, err := settings.ParseSchemas(settingsJson)
	if err != nil {
		return nil, err
	}
	return schemas, nil
}
