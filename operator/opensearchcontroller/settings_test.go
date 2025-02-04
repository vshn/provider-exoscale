package opensearchcontroller

import (
	"context"
	"testing"

	exoscalesdk "github.com/exoscale/egoscale/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type fakeSettingsFetcher struct{}

func (fakeSettingsFetcher) GetDBAASSettingsOpensearch(ctx context.Context) (*exoscalesdk.GetDBAASSettingsOpensearchResponse, error) {
	return &exoscalesdk.GetDBAASSettingsOpensearchResponse{
		Settings: &opensearchSettings,
	}, nil
}

func mustToRawExt(t *testing.T, set map[string]interface{}) runtime.RawExtension {
	res, err := mapper.ToRawExtension(&set)
	require.NoError(t, err, "failed to parse input setting")
	return res
}

func TestDefaultSettings(t *testing.T) {
	found := exoscalev1.OpenSearchParameters{
		Maintenance: exoscalev1.MaintenanceSpec{},
		Zone:        "gva-2",
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: false,
			Size: exoscalev1.SizeSpec{
				Plan: "startup-4",
			},
		},
		OpenSearchSettings: mustToRawExt(t, map[string]interface{}{
			"thread_pool_search_throttled_size": 42,
		}),
	}

	withDefaults, err := setSettingsDefaults(context.Background(), fakeSettingsFetcher{}, &found)
	require.NoError(t, err, "failed to set defaults")
	settingsWithDefaults, err := mapper.ToMap(withDefaults.OpenSearchSettings)
	require.NoError(t, err, "failed to parse set defaults")
	assert.EqualValues(t, 42, settingsWithDefaults["thread_pool_search_throttled_size"])
	assert.Len(t, settingsWithDefaults, 1)
}

var opensearchSettings = exoscalesdk.GetDBAASSettingsOpensearchResponseSettings{
	Opensearch: &exoscalesdk.GetDBAASSettingsOpensearchResponseSettingsOpensearch{
		Properties: map[string]any{
			"thread_pool_search_throttled_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "search_throttled thread pool size",
				"minimum":     1,
			},
			"thread_pool_analyze_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "analyze thread pool size",
				"minimum":     1,
			},
			"thread_pool_get_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "get thread pool size",
				"minimum":     1,
			},
			"thread_pool_get_queue_size": map[string]any{
				"description": "Size for the thread pool queue. See documentation for exact details.",
				"maximum":     2000,
				"type":        "integer",
				"title":       "get thread pool queue size",
				"minimum":     10,
			},
			"indices_recovery_max_concurrent_file_chunks": map[string]any{
				"description": "Number of file chunks sent in parallel for each recovery. Defaults to 2.",
				"maximum":     5,
				"type":        "integer",
				"title":       "indices.recovery.max_concurrent_file_chunks",
				"minimum":     2,
			},
			"indices_queries_cache_size": map[string]any{
				"description": "Percentage value. Default is 10%. Maximum amount of heap used for query cache. This is an expert setting. Too low value will decrease query performance and increase performance for other operations; too high value will cause issues with other OpenSearch functionality.",
				"maximum":     40,
				"type":        "integer",
				"title":       "indices.queries.cache.size",
				"minimum":     3,
			},
			"thread_pool_search_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "search thread pool size",
				"minimum":     1,
			},
			"indices_recovery_max_bytes_per_sec": map[string]any{
				"description": "Limits total inbound and outbound recovery traffic for each node. Applies to both peer recoveries as well as snapshot recoveries (i.e., restores from a snapshot). Defaults to 40mb",
				"maximum":     400,
				"type":        "integer",
				"title":       "indices.recovery.max_bytes_per_sec",
				"minimum":     40,
			},
			"http_max_initial_line_length": map[string]any{
				"description": "The max length of an HTTP URL, in bytes",
				"maximum":     65536,
				"type":        "integer",
				"title":       "http.max_initial_line_length",
				"minimum":     1024,
				"example":     4096,
			},
			"thread_pool_write_queue_size": map[string]any{
				"description": "Size for the thread pool queue. See documentation for exact details.",
				"maximum":     2000,
				"type":        "integer",
				"title":       "write thread pool queue size",
				"minimum":     10,
			},
			"script_max_compilations_rate": map[string]any{
				"description": "Script compilation circuit breaker limits the number of inline script compilations within a period of time. Default is use-context",
				"type":        "string",
				"title":       "Script max compilation rate - circuit breaker to prevent/minimize OOMs",
				"maxLength":   1024,
				"example":     "75/5m",
			},
			"search_max_buckets": map[string]any{
				"description": "Maximum number of aggregation buckets allowed in a single response. OpenSearch default value is used when this is not defined.",
				"maximum":     20000,
				"type": []string{
					"integer",
					"null",
				},
				"title":   "search.max_buckets",
				"minimum": 1,
				"example": 10000,
			},
			"reindex_remote_whitelist": map[string]any{
				"description": "Whitelisted addresses for reindexing. Changing this value will cause all OpenSearch instances to restart.",
				"type": []string{
					"array",
					"null",
				},
				"title": "reindex_remote_whitelist",
				"items": map[string]any{
					"type": []string{
						"string",
						"null",
					},
					"title":     "Address (hostname:port or IP:port)",
					"maxLength": 261,
					"example":   "anotherservice.aivencloud.com:12398",
				},
				"maxItems": 32,
			},
			"override_main_response_version": map[string]any{
				"description": "Compatibility mode sets OpenSearch to report its version as 7.10 so clients continue to work. Default is false",
				"type":        "boolean",
				"title":       "compatibility.override_main_response_version",
				"example":     true,
			},
			"http_max_header_size": map[string]any{
				"description": "The max size of allowed headers, in bytes",
				"maximum":     262144,
				"type":        "integer",
				"title":       "http.max_header_size",
				"minimum":     1024,
				"example":     8192,
			},
			"email_sender_name": map[string]any{
				"description": "This should be identical to the Sender name defined in Opensearch dashboards",
				"type": []string{
					"string",
				},
				"user_error": "Must consist of lower-case alpha-numeric characters and dashes, max 40 characters",
				"title":      "Sender email name placeholder to be used in Opensearch Dashboards and Opensearch keystore",
				"maxLength":  40,
				"example":    "alert-sender",
				"pattern":    "^[a-zA-Z0-9-_]+$",
			},
			"indices_fielddata_cache_size": map[string]any{
				"description": "Relative amount. Maximum amount of heap memory used for field data cache. This is an expert setting; decreasing the value too much will increase overhead of loading field data; too much memory used for field data cache will decrease amount of heap available for other operations.",
				"default":     "null",
				"maximum":     100,
				"type": []string{
					"integer",
					"null",
				},
				"title":   "indices.fielddata.cache.size",
				"minimum": 3,
			},
			"action_destructive_requires_name": map[string]any{
				"type": []string{
					"boolean",
					"null",
				},
				"title":   "Require explicit index names when deleting",
				"example": true,
			},
			"email_sender_username": map[string]any{
				"type": []string{
					"string",
				},
				"user_error": "Must be a valid email address",
				"title":      "Sender email address for Opensearch alerts",
				"maxLength":  320,
				"example":    "jane@example.com",
				"pattern":    "^[A-Za-z0-9_\\-\\.+\\'&]+@(([\\da-zA-Z])([_\\w-]{,62})\\.){,127}(([\\da-zA-Z])[_\\w-]{,61})?([\\da-zA-Z]\\.((xn\\-\\-[a-zA-Z\\d]+)|([a-zA-Z\\d]{2,})))$",
			},
			"indices_memory_index_buffer_size": map[string]any{
				"description": "Percentage value. Default is 10%. Total amount of heap used for indexing buffer, before writing segments to disk. This is an expert setting. Too low value will slow down indexing; too high value will increase indexing performance but causes performance issues for query performance.",
				"maximum":     40,
				"type":        "integer",
				"title":       "indices.memory.index_buffer_size",
				"minimum":     3,
			},
			"thread_pool_force_merge_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "force_merge thread pool size",
				"minimum":     1,
			},
			"cluster_routing_allocation_node_concurrent_recoveries": map[string]any{
				"description": "How many concurrent incoming/outgoing shard recoveries (normally replicas) are allowed to happen on a node. Defaults to 2.",
				"maximum":     16,
				"type":        "integer",
				"title":       "Concurrent incoming/outgoing shard recoveries per node",
				"minimum":     2,
			},
			"email_sender_password": map[string]any{
				"description": "Sender email password for Opensearch alerts to authenticate with SMTP server",
				"type": []string{
					"string",
				},
				"title":     "Sender email password for Opensearch alerts to authenticate with SMTP server",
				"maxLength": 1024,
				"example":   "very-secure-mail-password",
				"pattern":   "^[^\\x00-\\x1F]+$",
			},
			"thread_pool_analyze_queue_size": map[string]any{
				"description": "Size for the thread pool queue. See documentation for exact details.",
				"maximum":     2000,
				"type":        "integer",
				"title":       "analyze thread pool queue size",
				"minimum":     10,
			},
			"action_auto_create_index_enabled": map[string]any{
				"description": "Explicitly allow or block automatic creation of indices. Defaults to true",
				"type":        "boolean",
				"title":       "action.auto_create_index",
				"example":     false,
			},
			"http_max_content_length": map[string]any{
				"description": "Maximum content length for HTTP requests to the OpenSearch HTTP API, in bytes.",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "http.max_content_length",
				"minimum":     1,
			},
			"thread_pool_write_size": map[string]any{
				"description": "Size for the thread pool. See documentation for exact details. Do note this may have maximum value depending on CPU count - value is automatically lowered if set to higher than maximum value.",
				"maximum":     128,
				"type":        "integer",
				"title":       "write thread pool size",
				"minimum":     1,
			},
			"thread_pool_search_queue_size": map[string]any{
				"description": "Size for the thread pool queue. See documentation for exact details.",
				"maximum":     2000,
				"type":        "integer",
				"title":       "search thread pool queue size",
				"minimum":     10,
			},
			"indices_query_bool_max_clause_count": map[string]any{
				"description": "Maximum number of clauses Lucene BooleanQuery can have. The default value (1024) is relatively high, and increasing it may cause performance issues. Investigate other approaches first before increasing this value.",
				"maximum":     4096,
				"type":        "integer",
				"title":       "indices.query.bool.max_clause_count",
				"minimum":     64,
			},
			"thread_pool_search_throttled_queue_size": map[string]any{
				"description": "Size for the thread pool queue. See documentation for exact details.",
				"maximum":     2000,
				"type":        "integer",
				"title":       "search_throttled thread pool queue size",
				"minimum":     10,
			},
			"cluster_max_shards_per_node": map[string]any{
				"description": "Controls the number of shards allowed in the cluster per data node",
				"maximum":     10000,
				"type":        "integer",
				"title":       "cluster.max_shards_per_node",
				"minimum":     100,
				"example":     1000,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "OpenSearch settings",
	},
}
