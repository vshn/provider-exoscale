package rediscontroller

import (
	"context"
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
)

//nolint:golint,unused
var emptyRedisSettings = map[string]interface{}{
	"lfu_decay_time":         1,
	"ssl":                    true,
	"lfu_log_factor":         10,
	"notify_keyspace_events": "",
	"timeout":                300,
	"maxmemory_policy":       "noeviction",
}

type fakeSettingsFetcher struct{}

func (fakeSettingsFetcher) GetDBAASSettingsRedis(ctx context.Context) (*exoscalesdk.GetDBAASSettingsRedisResponse, error) {
	return &exoscalesdk.GetDBAASSettingsRedisResponse{
		Settings: &redisSettings,
	}, nil
}

func TestDefaultSettings(t *testing.T) {
	foundSettings := map[string]interface{}{
		"lfu_decay_time": 2,
		"persistence":    "rdb",
	}
	foundSettingRaw, err := mapper.ToRawExtension(&foundSettings)
	require.NoError(t, err, "failed to parse input setting")
	found := exoscalev1.RedisParameters{
		Maintenance: exoscalev1.MaintenanceSpec{},
		Zone:        "gva-2",
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: false,
			Size: exoscalev1.SizeSpec{
				Plan: "startup-4",
			},
		},
		RedisSettings: foundSettingRaw,
	}

	withDefaults, err := setSettingsDefaults(context.Background(), fakeSettingsFetcher{}, &found)
	require.NoError(t, err, "failed to set defaults")
	setingsWithDefaults, err := mapper.ToMap(withDefaults.RedisSettings)
	require.NoError(t, err, "failed to parse set defaults")
	assert.EqualValues(t, 2, setingsWithDefaults["lfu_decay_time"])
	assert.EqualValues(t, "rdb", setingsWithDefaults["persistence"])
	assert.EqualValues(t, true, setingsWithDefaults["ssl"])
	assert.EqualValues(t, 10, setingsWithDefaults["lfu_log_factor"])
	assert.EqualValues(t, "", setingsWithDefaults["notify_keyspace_events"])
	assert.EqualValues(t, 300, setingsWithDefaults["timeout"])
	assert.EqualValues(t, "noeviction", setingsWithDefaults["maxmemory_policy"])
}

var redisSettings = exoscalesdk.GetDBAASSettingsRedisResponseSettings{
	Redis: &exoscalesdk.GetDBAASSettingsRedisResponseSettingsRedis{
		Properties: map[string]any{
			"ssl": map[string]any{
				"default": true,
				"type":    "boolean",
				"title":   "Require SSL to access Redis",
			},
			"lfu_log_factor": map[string]any{
				"default": 10,
				"maximum": 100,
				"type":    "integer",
				"title":   "Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies",
				"minimum": 0,
			},
			"maxmemory_policy": map[string]any{
				"enum": []string{
					"noeviction",
					"allkeys-lru",
					"volatile-lru",
					"allkeys-random",
					"volatile-random",
					"volatile-ttl",
					"volatile-lfu",
					"allkeys-lfu",
				},
				"default": "noeviction",
				"type": []string{
					"string",
					"null",
				},
				"title": "Redis maxmemory-policy",
			},
			"io_threads": map[string]any{
				"maximum": 32,
				"type":    "integer",
				"title":   "Redis IO thread count",
				"minimum": 1,
				"example": 1,
			},
			"lfu_decay_time": map[string]any{
				"default": 1,
				"maximum": 120,
				"type":    "integer",
				"title":   "LFU maxmemory-policy counter decay time in minutes",
				"minimum": 1,
			},
			"pubsub_client_output_buffer_limit": map[string]any{
				"description": "Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.",
				"maximum":     512,
				"type":        "integer",
				"title":       "Pub/sub client output buffer hard limit in MB",
				"minimum":     32,
				"example":     64,
			},
			"notify_keyspace_events": map[string]any{
				"default":   "",
				"type":      "string",
				"title":     "Set notify-keyspace-events option",
				"maxLength": 32,
				"pattern":   "^[KEg\\$lshzxeA]*$",
			},
			"persistence": map[string]any{
				"description": "When persistence is 'rdb', Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to backup schedule for backup purposes. When persistence is 'off', no RDB dumps and backups are done, so data can be lost at any moment if service is restarted for any reason, or if service is powered off. Also service can't be forked.",
				"enum": []string{
					"off",
					"rdb",
				},
				"type":  "string",
				"title": "Redis persistence",
			},
			"timeout": map[string]any{
				"default": 300,
				"maximum": 31536000,
				"type":    "integer",
				"title":   "Redis idle connection timeout in seconds",
				"minimum": 0,
			},
			"acl_channels_default": map[string]any{
				"description": "Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default.",
				"enum": []string{
					"allchannels",
					"resetchannels",
				},
				"type":  "string",
				"title": "Default ACL for pub/sub channels used when Redis user is created",
			},
			"number_of_databases": map[string]any{
				"description": "Set number of redis databases. Changing this will cause a restart of redis service.",
				"maximum":     128,
				"type":        "integer",
				"title":       "Number of redis databases",
				"minimum":     1,
				"example":     16,
			},
		},
		Type:  "object",
		Title: "Redis settings",
	},
}
