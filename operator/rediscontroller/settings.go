package rediscontroller

import (
	"context"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDbaasSettingsRedisWithResponse(ctx context.Context, reqEditors ...oapi.RequestEditorFn) (*oapi.GetDbaasSettingsRedisResponse, error)
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
	resp, err := f.GetDbaasSettingsRedisWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	schemas, err := settings.ParseSchemas(resp.Body)
	if err != nil {
		return nil, err
	}
	return schemas, nil
}
