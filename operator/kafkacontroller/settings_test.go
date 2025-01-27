//go:build ignore

package kafkacontroller

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

//nolint:golint,unused
var emptyKafkaRestSettings = map[string]interface{}{
	"simpleconsumer_pool_size_max": 25,
	"producer_linger_ms":           0,
	"consumer_request_timeout_ms":  1000,
	"consumer_enable_auto_commit":  true,
	"producer_acks":                1,
	"consumer_request_max_bytes":   6.7108864e+07,
}

type fakeSettingsFetcher struct{}

func (fakeSettingsFetcher) GetDBAASSettingsKafka(ctx context.Context) (*exoscalesdk.GetDBAASSettingsKafkaResponse, error) {
	return &exoscalesdk.GetDBAASSettingsKafkaResponse{
		Settings: &kafkaSettings,
	}, nil
}

func mustToRawExt(t *testing.T, set map[string]interface{}) runtime.RawExtension {
	res, err := mapper.ToRawExtension(&set)
	require.NoError(t, err, "failed to parse input setting")
	return res
}

func mustToMap(t *testing.T, raw runtime.RawExtension) map[string]interface{} {
	res, err := mapper.ToMap(raw)
	require.NoError(t, err, "failed to parse set defaults")
	return res
}

func TestDefaultSettings(t *testing.T) {

	found := exoscalev1.KafkaParameters{
		Maintenance: exoscalev1.MaintenanceSpec{},
		Zone:        "gva-2",
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: false,
			Size: exoscalev1.SizeSpec{
				Plan: "startup-4",
			},
		},
		KafkaRestSettings: mustToRawExt(t, map[string]interface{}{
			"producer_compression_type": "gzip",
			"producer_acks":             4,
		}),
		KafkaSettings: mustToRawExt(t, map[string]interface{}{
			"group_max_session_timeout_ms": 42,
		}),
	}

	withDefaults, err := setSettingsDefaults(context.Background(), fakeSettingsFetcher{}, &found)
	require.NoError(t, err, "failed to set defaults")

	restWD := mustToMap(t, withDefaults.KafkaRestSettings)
	assert.EqualValues(t, 25, restWD["simpleconsumer_pool_size_max"])
	assert.EqualValues(t, "gzip", restWD["producer_compression_type"])
	assert.EqualValues(t, 4, restWD["producer_acks"])

	assert.EqualValues(t, 42, mustToMap(t, withDefaults.KafkaSettings)["group_max_session_timeout_ms"])
}

var kafkaSettings = exoscalesdk.GetDBAASSettingsKafkaResponseSettings{
	Kafka: &exoscalesdk.GetDBAASSettingsKafkaResponseSettingsKafka{
		Properties: map[string]any{
			"group_max_session_timeout_ms": map[string]any{
				"description": "The maximum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.",
				"maximum":     1800000,
				"type":        "integer",
				"title":       "group.max.session.timeout.ms",
				"minimum":     0,
				"example":     1800000,
			},
			"log_flush_interval_messages": map[string]any{
				"description": "The number of messages accumulated on a log partition before messages are flushed to disk",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.flush.interval.messages",
				"minimum":     1,
				"example":     9223372036854775807,
			},
			"max_connections_per_ip": map[string]any{
				"description": "The maximum number of connections allowed from each ip address (defaults to 2147483647).",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "max.connections.per.ip",
				"minimum":     256,
			},
			"log_index_size_max_bytes": map[string]any{
				"description": "The maximum size in bytes of the offset index",
				"maximum":     104857600,
				"type":        "integer",
				"title":       "log.index.size.max.bytes",
				"minimum":     1048576,
				"example":     10485760,
			},
			"auto_create_topics_enable": map[string]any{
				"description": "Enable auto creation of topics",
				"type":        "boolean",
				"title":       "auto.create.topics.enable",
				"example":     true,
			},
			"log_index_interval_bytes": map[string]any{
				"description": "The interval with which Kafka adds an entry to the offset index",
				"maximum":     104857600,
				"type":        "integer",
				"title":       "log.index.interval.bytes",
				"minimum":     0,
				"example":     4096,
			},
			"replica_fetch_max_bytes": map[string]any{
				"description": "The number of bytes of messages to attempt to fetch for each partition (defaults to 1048576). This is not an absolute maximum, if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made.",
				"maximum":     104857600,
				"type":        "integer",
				"title":       "replica.fetch.max.bytes",
				"minimum":     1048576,
			},
			"num_partitions": map[string]any{
				"description": "Number of partitions for autocreated topics",
				"maximum":     1000,
				"type":        "integer",
				"title":       "num.partitions",
				"minimum":     1,
			},
			"transaction_state_log_segment_bytes": map[string]any{
				"description": "The transaction topic segment bytes should be kept relatively small in order to facilitate faster log compaction and cache loads (defaults to 104857600 (100 mebibytes)).",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "transaction.state.log.segment.bytes",
				"minimum":     1048576,
				"example":     104857600,
			},
			"replica_fetch_response_max_bytes": map[string]any{
				"description": "Maximum bytes expected for the entire fetch response (defaults to 10485760). Records are fetched in batches, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that progress can be made. As such, this is not an absolute maximum.",
				"maximum":     1048576000,
				"type":        "integer",
				"title":       "replica.fetch.response.max.bytes",
				"minimum":     10485760,
			},
			"log_message_timestamp_type": map[string]any{
				"description": "Define whether the timestamp in the message is message create time or log append time.",
				"enum": []string{
					"CreateTime",
					"LogAppendTime",
				},
				"type":  "string",
				"title": "log.message.timestamp.type",
			},
			"connections_max_idle_ms": map[string]any{
				"description": "Idle connections timeout: the server socket processor threads close the connections that idle for longer than this.",
				"maximum":     3600000,
				"type":        "integer",
				"title":       "connections.max.idle.ms",
				"minimum":     1000,
				"example":     540000,
			},
			"log_flush_interval_ms": map[string]any{
				"description": "The maximum time in ms that a message in any topic is kept in memory before flushed to disk. If not set, the value in log.flush.scheduler.interval.ms is used",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.flush.interval.ms",
				"minimum":     0,
			},
			"log_preallocate": map[string]any{
				"description": "Should pre allocate file when create new segment?",
				"type":        "boolean",
				"title":       "log.preallocate",
				"example":     false,
			},
			"log_segment_delete_delay_ms": map[string]any{
				"description": "The amount of time to wait before deleting a file from the filesystem",
				"maximum":     3600000,
				"type":        "integer",
				"title":       "log.segment.delete.delay.ms",
				"minimum":     0,
				"example":     60000,
			},
			"message_max_bytes": map[string]any{
				"description": "The maximum size of message that the server can receive.",
				"maximum":     100001200,
				"type":        "integer",
				"title":       "message.max.bytes",
				"minimum":     0,
				"example":     1048588,
			},
			"log_cleaner_min_cleanable_ratio": map[string]any{
				"description": "Controls log compactor frequency. Larger value means more frequent compactions but also more space wasted for logs. Consider setting log.cleaner.max.compaction.lag.ms to enforce compactions sooner, instead of setting a very high value for this option.",
				"maximum":     0.9,
				"type":        "number",
				"title":       "log.cleaner.min.cleanable.ratio",
				"minimum":     0.2,
				"example":     0.5,
			},
			"group_initial_rebalance_delay_ms": map[string]any{
				"description": "The amount of time, in milliseconds, the group coordinator will wait for more consumers to join a new group before performing the first rebalance. A longer delay means potentially fewer rebalances, but increases the time until processing begins. The default value for this is 3 seconds. During development and testing it might be desirable to set this to 0 in order to not delay test execution time.",
				"maximum":     300000,
				"type":        "integer",
				"title":       "group.initial.rebalance.delay.ms",
				"minimum":     0,
				"example":     3000,
			},
			"log_cleanup_policy": map[string]any{
				"description": "The default cleanup policy for segments beyond the retention window",
				"enum": []string{
					"delete",
					"compact",
					"compact,delete",
				},
				"type":    "string",
				"title":   "log.cleanup.policy",
				"example": "delete",
			},
			"log_roll_jitter_ms": map[string]any{
				"description": "The maximum jitter to subtract from logRollTimeMillis (in milliseconds). If not set, the value in log.roll.jitter.hours is used",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.roll.jitter.ms",
				"minimum":     0,
			},
			"transaction_remove_expired_transaction_cleanup_interval_ms": map[string]any{
				"description": "The interval at which to remove transactions that have expired due to transactional.id.expiration.ms passing (defaults to 3600000 (1 hour)).",
				"maximum":     3600000,
				"type":        "integer",
				"title":       "transaction.remove.expired.transaction.cleanup.interval.ms",
				"minimum":     600000,
				"example":     3600000,
			},
			"default_replication_factor": map[string]any{
				"description": "Replication factor for autocreated topics",
				"maximum":     10,
				"type":        "integer",
				"title":       "default.replication.factor",
				"minimum":     1,
			},
			"log_roll_ms": map[string]any{
				"description": "The maximum time before a new log segment is rolled out (in milliseconds).",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.roll.ms",
				"minimum":     1,
			},
			"producer_purgatory_purge_interval_requests": map[string]any{
				"description": "The purge interval (in number of requests) of the producer request purgatory(defaults to 1000).",
				"maximum":     10000,
				"type":        "integer",
				"title":       "producer.purgatory.purge.interval.requests",
				"minimum":     10,
			},
			"log_retention_bytes": map[string]any{
				"description": "The maximum size of the log before deleting messages",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.retention.bytes",
				"minimum":     -1,
			},
			"log_cleaner_min_compaction_lag_ms": map[string]any{
				"description": "The minimum time a message will remain uncompacted in the log. Only applicable for logs that are being compacted.",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.cleaner.min.compaction.lag.ms",
				"minimum":     0,
			},
			"min_insync_replicas": map[string]any{
				"description": "When a producer sets acks to 'all' (or '-1'), min.insync.replicas specifies the minimum number of replicas that must acknowledge a write for the write to be considered successful.",
				"maximum":     7,
				"type":        "integer",
				"title":       "min.insync.replicas",
				"minimum":     1,
				"example":     1,
			},
			"compression_type": map[string]any{
				"description": "Specify the final compression type for a given topic. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'uncompressed' which is equivalent to no compression; and 'producer' which means retain the original compression codec set by the producer.",
				"enum": []string{
					"gzip",
					"snappy",
					"lz4",
					"zstd",
					"uncompressed",
					"producer",
				},
				"type":  "string",
				"title": "compression.type",
			},
			"log_message_timestamp_difference_max_ms": map[string]any{
				"description": "The maximum difference allowed between the timestamp when a broker receives a message and the timestamp specified in the message",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.message.timestamp.difference.max.ms",
				"minimum":     0,
			},
			"log_message_downconversion_enable": map[string]any{
				"description": "This configuration controls whether down-conversion of message formats is enabled to satisfy consume requests. ",
				"type":        "boolean",
				"title":       "log.message.downconversion.enable",
				"example":     true,
			},
			"max_incremental_fetch_session_cache_slots": map[string]any{
				"description": "The maximum number of incremental fetch sessions that the broker will maintain.",
				"maximum":     10000,
				"type":        "integer",
				"title":       "max.incremental.fetch.session.cache.slots",
				"minimum":     1000,
				"example":     1000,
			},
			"log_cleaner_max_compaction_lag_ms": map[string]any{
				"description": "The maximum amount of time message will remain uncompacted. Only applicable for logs that are being compacted",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.cleaner.max.compaction.lag.ms",
				"minimum":     30000,
			},
			"log_retention_hours": map[string]any{
				"description": "The number of hours to keep a log file before deleting it",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "log.retention.hours",
				"minimum":     -1,
			},
			"group_min_session_timeout_ms": map[string]any{
				"description": "The minimum allowed session timeout for registered consumers. Longer timeouts give consumers more time to process messages in between heartbeats at the cost of a longer time to detect failures.",
				"maximum":     60000,
				"type":        "integer",
				"title":       "group.min.session.timeout.ms",
				"minimum":     0,
				"example":     6000,
			},
			"socket_request_max_bytes": map[string]any{
				"description": "The maximum number of bytes in a socket request (defaults to 104857600).",
				"maximum":     209715200,
				"type":        "integer",
				"title":       "socket.request.max.bytes",
				"minimum":     10485760,
			},
			"log_cleaner_delete_retention_ms": map[string]any{
				"description": "How long are delete records retained?",
				"maximum":     315569260000,
				"type":        "integer",
				"title":       "log.cleaner.delete.retention.ms",
				"minimum":     0,
				"example":     86400000,
			},
			"log_segment_bytes": map[string]any{
				"description": "The maximum size of a single log file",
				"maximum":     1073741824,
				"type":        "integer",
				"title":       "log.segment.bytes",
				"minimum":     10485760,
			},
			"offsets_retention_minutes": map[string]any{
				"description": "Log retention window in minutes for offsets topic",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "offsets.retention.minutes",
				"minimum":     1,
				"example":     10080,
			},
			"log_retention_ms": map[string]any{
				"description": "The number of milliseconds to keep a log file before deleting it (in milliseconds), If not set, the value in log.retention.minutes is used. If set to -1, no time limit is applied.",
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "log.retention.ms",
				"minimum":     -1,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "Kafka broker configuration values",
	},
	KafkaConnect: &exoscalesdk.GetDBAASSettingsKafkaResponseSettingsKafkaConnect{
		Properties: map[string]any{
			"producer_buffer_memory": map[string]any{
				"description": "The total bytes of memory the producer can use to buffer records waiting to be sent to the broker (defaults to 33554432).",
				"maximum":     134217728,
				"type":        "integer",
				"title":       "The total bytes of memory the producer can use to buffer records waiting to be sent to the broker",
				"minimum":     5242880,
				"example":     8388608,
			},
			"consumer_max_poll_interval_ms": map[string]any{
				"description": "The maximum delay in milliseconds between invocations of poll() when using consumer group management (defaults to 300000).",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "The maximum delay between polls when using consumer group management",
				"minimum":     1,
				"example":     300000,
			},
			"producer_compression_type": map[string]any{
				"description": "Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression.",
				"enum": []string{
					"gzip",
					"snappy",
					"lz4",
					"zstd",
					"none",
				},
				"type":  "string",
				"title": "The default compression type for producers",
			},
			"connector_client_config_override_policy": map[string]any{
				"description": "Defines what client configurations can be overridden by the connector. Default is None",
				"enum": []string{
					"None",
					"All",
				},
				"type":  "string",
				"title": "Client config override policy",
			},
			"offset_flush_interval_ms": map[string]any{
				"description": "The interval at which to try committing offsets for tasks (defaults to 60000).",
				"maximum":     100000000,
				"type":        "integer",
				"title":       "The interval at which to try committing offsets for tasks",
				"minimum":     1,
				"example":     60000,
			},
			"consumer_fetch_max_bytes": map[string]any{
				"description": "Records are fetched in batches by the consumer, and if the first record batch in the first non-empty partition of the fetch is larger than this value, the record batch will still be returned to ensure that the consumer can make progress. As such, this is not a absolute maximum.",
				"maximum":     104857600,
				"type":        "integer",
				"title":       "The maximum amount of data the server should return for a fetch request",
				"minimum":     1048576,
				"example":     52428800,
			},
			"consumer_max_partition_fetch_bytes": map[string]any{
				"description": "Records are fetched in batches by the consumer.If the first record batch in the first non-empty partition of the fetch is larger than this limit, the batch will still be returned to ensure that the consumer can make progress. ",
				"maximum":     104857600,
				"type":        "integer",
				"title":       "The maximum amount of data per-partition the server will return.",
				"minimum":     1048576,
				"example":     1048576,
			},
			"offset_flush_timeout_ms": map[string]any{
				"description": "Maximum number of milliseconds to wait for records to flush and partition offset data to be committed to offset storage before cancelling the process and restoring the offset data to be committed in a future attempt (defaults to 5000).",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "Offset flush timeout",
				"minimum":     1,
				"example":     5000,
			},
			"consumer_auto_offset_reset": map[string]any{
				"description": "What to do when there is no initial offset in Kafka or if the current offset does not exist any more on the server. Default is earliest",
				"enum": []string{
					"earliest",
					"latest",
				},
				"type":  "string",
				"title": "Consumer auto offset reset",
			},
			"producer_max_request_size": map[string]any{
				"description": "This setting will limit the number of record batches the producer will send in a single request to avoid sending huge requests.",
				"maximum":     67108864,
				"type":        "integer",
				"title":       "The maximum size of a request in bytes",
				"minimum":     131072,
				"example":     1048576,
			},
			"producer_batch_size": map[string]any{
				"description": "This setting gives the upper bound of the batch size to be sent. If there are fewer than this many bytes accumulated for this partition, the producer will 'linger' for the linger.ms time waiting for more records to show up. A batch size of zero will disable batching entirely (defaults to 16384).",
				"maximum":     5242880,
				"type":        "integer",
				"title":       "The batch size in bytes the producer will attempt to collect for the same partition before publishing to broker",
				"minimum":     0,
				"example":     1024,
			},
			"session_timeout_ms": map[string]any{
				"description": "The timeout in milliseconds used to detect failures when using Kafka’s group management facilities (defaults to 10000).",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "The timeout used to detect failures when using Kafka’s group management facilities",
				"minimum":     1,
				"example":     10000,
			},
			"producer_linger_ms": map[string]any{
				"description": "This setting gives the upper bound on the delay for batching: once there is batch.size worth of records for a partition it will be sent immediately regardless of this setting, however if there are fewer than this many bytes accumulated for this partition the producer will 'linger' for the specified time waiting for more records to show up. Defaults to 0.",
				"maximum":     5000,
				"type":        "integer",
				"title":       "Wait for up to the given delay to allow batching records together",
				"minimum":     0,
				"example":     100,
			},
			"consumer_isolation_level": map[string]any{
				"description": "Transaction read isolation level. read_uncommitted is the default, but read_committed can be used if consume-exactly-once behavior is desired.",
				"enum": []string{
					"read_uncommitted",
					"read_committed",
				},
				"type":  "string",
				"title": "Consumer isolation level",
			},
			"consumer_max_poll_records": map[string]any{
				"description": "The maximum number of records returned in a single call to poll() (defaults to 500).",
				"maximum":     10000,
				"type":        "integer",
				"title":       "The maximum number of records returned by a single poll",
				"minimum":     1,
				"example":     500,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "Kafka Connect configuration values",
	},
	KafkaRest: &exoscalesdk.GetDBAASSettingsKafkaResponseSettingsKafkaRest{
		Properties: map[string]any{
			"producer_compression_type": map[string]any{
				"description": "Specify the default compression type for producers. This configuration accepts the standard compression codecs ('gzip', 'snappy', 'lz4', 'zstd'). It additionally accepts 'none' which is the default and equivalent to no compression.",
				"enum": []string{
					"gzip",
					"snappy",
					"lz4",
					"zstd",
					"none",
				},
				"type":  "string",
				"title": "producer.compression.type",
			},
			"consumer_enable_auto_commit": map[string]any{
				"description": "If true the consumer's offset will be periodically committed to Kafka in the background",
				"default":     true,
				"type":        "boolean",
				"title":       "consumer.enable.auto.commit",
			},
			"producer_acks": map[string]any{
				"description": "The number of acknowledgments the producer requires the leader to have received before considering a request complete. If set to 'all' or '-1', the leader will wait for the full set of in-sync replicas to acknowledge the record.",
				"enum": []string{
					"all",
					"-1",
					"0",
					"1",
				},
				"default": "1",
				"type":    "string",
				"title":   "producer.acks",
			},
			"consumer_request_max_bytes": map[string]any{
				"description": "Maximum number of bytes in unencoded message keys and values by a single request",
				"default":     67108864,
				"maximum":     671088640,
				"type":        "integer",
				"title":       "consumer.request.max.bytes",
				"minimum":     0,
			},
			"simpleconsumer_pool_size_max": map[string]any{
				"description": "Maximum number of SimpleConsumers that can be instantiated per broker",
				"default":     25,
				"maximum":     250,
				"type":        "integer",
				"title":       "simpleconsumer.pool.size.max",
				"minimum":     10,
			},
			"producer_linger_ms": map[string]any{
				"description": "Wait for up to the given delay to allow batching records together",
				"default":     0,
				"maximum":     5000,
				"type":        "integer",
				"title":       "producer.linger.ms",
				"minimum":     0,
			},
			"consumer_request_timeout_ms": map[string]any{
				"description": "The maximum total time to wait for messages for a request if the maximum number of messages has not yet been reached",
				"enum": []int{
					1000,
					15000,
					30000,
				},
				"default": 1000,
				"maximum": 30000,
				"type":    "integer",
				"title":   "consumer.request.timeout.ms",
				"minimum": 1000,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "Kafka REST configuration",
	},
	SchemaRegistry: &exoscalesdk.GetDBAASSettingsKafkaResponseSettingsSchemaRegistry{
		Properties: map[string]any{
			"topic_name": map[string]any{
				"description": "The durable single partition topic that acts as the durable log for the data. This topic must be compacted to avoid losing data due to retention policy. Please note that changing this configuration in an existing Schema Registry / Karapace setup leads to previous schemas being inaccessible, data encoded with them potentially unreadable and schema ID sequence put out of order. It's only possible to do the switch while Schema Registry / Karapace is disabled. Defaults to _schemas.",
				"type":        "string",
				"minLength":   1,
				"user_error":  "Must consist of alpha-numeric characters, underscores, dashes or dots, max 249 characters",
				"title":       "topic_name",
				"maxLength":   249,
				"example":     "_schemas",
				"pattern":     "^(?!\\.$|\\.\\.$)[-_.A-Za-z0-9]+$",
			},
			"leader_eligibility": map[string]any{
				"description": "If true, Karapace / Schema Registry on the service nodes can participate in leader election. It might be needed to disable this when the schemas topic is replicated to a secondary cluster and Karapace / Schema Registry there must not participate in leader election. Defaults to true.",
				"type":        "boolean",
				"title":       "leader_eligibility",
				"example":     true,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "Schema Registry configuration",
	},
}
