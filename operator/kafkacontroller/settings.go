package kafkacontroller

import (
	"context"

	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDbaasSettingsKafkaWithResponse(ctx context.Context, reqEditors ...oapi.RequestEditorFn) (*oapi.GetDbaasSettingsKafkaResponse, error)
}

func setSettingsDefaults(ctx context.Context, f settingsFetcher, in *exoscalev1.KafkaParameters) (*exoscalev1.KafkaParameters, error) {
	s, err := fetchSettingSchema(ctx, f)
	if err != nil {
		return nil, err
	}
	res := in.DeepCopy()

	res.KafkaRestSettings, err = s.SetDefaults("kafka-rest", res.KafkaRestSettings)
	if err != nil {
		return nil, err
	}
	res.KafkaSettings, err = s.SetDefaults("kafka", res.KafkaSettings)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func fetchSettingSchema(ctx context.Context, f settingsFetcher) (settings.Schemas, error) {
	resp, err := f.GetDbaasSettingsKafkaWithResponse(ctx)
	if err != nil {
		return nil, err
	}
	schemas, err := settings.ParseSchemas(resp.Body)
	if err != nil {
		return nil, err
	}
	return schemas, nil
}
