package opensearchcontroller

import (
	"context"
	"encoding/json"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDBAASSettingsOpensearch(ctx context.Context) (*exoscalesdk.GetDBAASSettingsOpensearchResponse, error)
}

func setSettingsDefaults(ctx context.Context, f settingsFetcher, in *exoscalev1.OpenSearchParameters) (*exoscalev1.OpenSearchParameters, error) {
	s, err := fetchSettingSchema(ctx, f)
	if err != nil {
		return nil, err
	}
	res := in.DeepCopy()

	res.OpenSearchSettings, err = s.SetDefaults("opensearch", res.OpenSearchSettings)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func fetchSettingSchema(ctx context.Context, f settingsFetcher) (settings.Schemas, error) {
	resp, err := f.GetDBAASSettingsOpensearch(ctx)
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
