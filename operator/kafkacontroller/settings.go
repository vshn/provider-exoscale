package kafkacontroller

import (
	"context"
	"encoding/json"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"

	"github.com/vshn/provider-exoscale/internal/settings"
)

type settingsFetcher interface {
	GetDBAASSettingsKafka(ctx context.Context) (*exoscalesdk.GetDBAASSettingsKafkaResponse, error)
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
	resp, err := f.GetDBAASSettingsKafka(ctx)
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
