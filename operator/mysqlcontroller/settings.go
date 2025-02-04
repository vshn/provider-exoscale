package mysqlcontroller

import (
	"context"
	"encoding/json"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDBAASSettingsMysql(ctx context.Context) (*exoscalesdk.GetDBAASSettingsMysqlResponse, error)
}

func setSettingsDefaults(ctx context.Context, f settingsFetcher, in *exoscalev1.MySQLParameters) (*exoscalev1.MySQLParameters, error) {
	s, err := fetchSettingSchema(ctx, f)
	if err != nil {
		return nil, err
	}
	res := in.DeepCopy()

	res.MySQLSettings, err = s.SetDefaults("mysql", res.MySQLSettings)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func fetchSettingSchema(ctx context.Context, f settingsFetcher) (settings.Schemas, error) {
	resp, err := f.GetDBAASSettingsMysql(ctx)
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
