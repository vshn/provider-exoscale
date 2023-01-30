package mysqlcontroller

import (
	"context"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDbaasSettingsMysqlWithResponse(ctx context.Context, reqEditors ...oapi.RequestEditorFn) (*oapi.GetDbaasSettingsMysqlResponse, error)
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
	resp, err := f.GetDbaasSettingsMysqlWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	schemas, err := settings.ParseSchemas(resp.Body)
	if err != nil {
		return nil, err
	}
	return schemas, nil
}
