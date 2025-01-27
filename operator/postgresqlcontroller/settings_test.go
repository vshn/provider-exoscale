package postgresqlcontroller

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

func (fakeSettingsFetcher) GetDBAASSettingsPG(ctx context.Context) (*exoscalesdk.GetDBAASSettingsPGResponse, error) {
	return &exoscalesdk.GetDBAASSettingsPGResponse{
		Settings: &pgSettings,
	}, nil
}

func mustToRawExt(t *testing.T, set map[string]interface{}) runtime.RawExtension {
	res, err := mapper.ToRawExtension(&set)
	require.NoError(t, err, "failed to parse input setting")
	return res
}

func TestDefaultSettings(t *testing.T) {
	found := exoscalev1.PostgreSQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{},
		Zone:        "gva-2",
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: false,
			Size: exoscalev1.SizeSpec{
				Plan: "startup-4",
			},
		},
		PGSettings: mustToRawExt(t, map[string]interface{}{
			"track_activity_query_size": 1025,
		}),
	}

	withDefaults, err := setSettingsDefaults(context.Background(), fakeSettingsFetcher{}, &found)
	require.NoError(t, err, "failed to set defaults")
	setingsWithDefaults, err := mapper.ToMap(withDefaults.PGSettings)
	require.NoError(t, err, "failed to parse set defaults")
	assert.EqualValues(t, 1025, setingsWithDefaults["track_activity_query_size"])
	assert.Len(t, setingsWithDefaults, 1)
}

var pgSettings = exoscalesdk.GetDBAASSettingsPGResponseSettings{
	PG: &exoscalesdk.GetDBAASSettingsPGResponseSettingsPG{
		Properties: map[string]any{
			"track_activity_query_size": map[string]any{
				"description": "Specifies the number of bytes reserved to track the currently executing command for each active session.",
				"maximum":     10240,
				"type":        "integer",
				"title":       "track_activity_query_size",
				"minimum":     1024,
				"example":     1024,
			},
			"log_autovacuum_min_duration": map[string]any{
				"description": "Causes each action executed by autovacuum to be logged if it ran for at least the specified number of milliseconds. Setting this to zero logs all autovacuum actions. Minus-one (the default) disables logging autovacuum actions.",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "log_autovacuum_min_duration",
				"minimum":     -1,
			},
			"autovacuum_vacuum_cost_limit": map[string]any{
				"description": "Specifies the cost limit value that will be used in automatic VACUUM operations. If -1 is specified (which is the default), the regular vacuum_cost_limit value will be used.",
				"maximum":     10000,
				"type":        "integer",
				"title":       "autovacuum_vacuum_cost_limit",
				"minimum":     -1,
			},
			"timezone": map[string]any{
				"description": "PostgreSQL service timezone",
				"type":        "string",
				"title":       "timezone",
				"maxLength":   64,
				"example":     "Europe/Helsinki",
			},
			"track_io_timing": map[string]any{
				"description": "Enables timing of database I/O calls. This parameter is off by default, because it will repeatedly query the operating system for the current time, which may cause significant overhead on some platforms.",
				"enum": []string{
					"off",
					"on",
				},
				"type":    "string",
				"title":   "track_io_timing",
				"example": "off",
			},
			"pg_stat_monitor.pgsm_enable_query_plan": map[string]any{
				"description": "Enables or disables query plan monitoring",
				"type":        "boolean",
				"title":       "pg_stat_monitor.pgsm_enable_query_plan",
				"example":     false,
			},
			"max_files_per_process": map[string]any{
				"description": "PostgreSQL maximum number of files that can be open per process",
				"maximum":     4096,
				"type":        "integer",
				"title":       "max_files_per_process",
				"minimum":     1000,
			},
			"pg_stat_monitor.pgsm_max_buckets": map[string]any{
				"description": "Sets the maximum number of buckets ",
				"maximum":     10,
				"type":        "integer",
				"title":       "pg_stat_monitor.pgsm_max_buckets",
				"minimum":     1,
				"example":     10,
			},
			"bgwriter_delay": map[string]any{
				"description": "Specifies the delay between activity rounds for the background writer in milliseconds. Default is 200.",
				"maximum":     10000,
				"type":        "integer",
				"title":       "bgwriter_delay",
				"minimum":     10,
				"example":     200,
			},
			"autovacuum_max_workers": map[string]any{
				"description": "Specifies the maximum number of autovacuum processes (other than the autovacuum launcher) that may be running at any one time. The default is three. This parameter can only be set at server start.",
				"maximum":     20,
				"type":        "integer",
				"title":       "autovacuum_max_workers",
				"minimum":     1,
			},
			"bgwriter_flush_after": map[string]any{
				"description": "Whenever more than bgwriter_flush_after bytes have been written by the background writer, attempt to force the OS to issue these writes to the underlying storage. Specified in kilobytes, default is 512. Setting of 0 disables forced writeback.",
				"maximum":     2048,
				"type":        "integer",
				"title":       "bgwriter_flush_after",
				"minimum":     0,
				"example":     512,
			},
			"default_toast_compression": map[string]any{
				"description": "Specifies the default TOAST compression method for values of compressible columns (the default is lz4).",
				"enum": []string{
					"lz4",
					"pglz",
				},
				"type":    "string",
				"title":   "default_toast_compression",
				"example": "lz4",
			},
			"deadlock_timeout": map[string]any{
				"description": "This is the amount of time, in milliseconds, to wait on a lock before checking to see if there is a deadlock condition.",
				"maximum":     1800000,
				"type":        "integer",
				"title":       "deadlock_timeout",
				"minimum":     500,
				"example":     1000,
			},
			"idle_in_transaction_session_timeout": map[string]any{
				"description": "Time out sessions with open transactions after this number of milliseconds",
				"maximum":     604800000,
				"type":        "integer",
				"title":       "idle_in_transaction_session_timeout",
				"minimum":     0,
			},
			"max_pred_locks_per_transaction": map[string]any{
				"description": "PostgreSQL maximum predicate locks per transaction",
				"maximum":     5120,
				"type":        "integer",
				"title":       "max_pred_locks_per_transaction",
				"minimum":     64,
			},
			"max_replication_slots": map[string]any{
				"description": "PostgreSQL maximum replication slots",
				"maximum":     64,
				"type":        "integer",
				"title":       "max_replication_slots",
				"minimum":     8,
			},
			"autovacuum_vacuum_threshold": map[string]any{
				"description": "Specifies the minimum number of updated or deleted tuples needed to trigger a VACUUM in any one table. The default is 50 tuples",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "autovacuum_vacuum_threshold",
				"minimum":     0,
			},
			"max_parallel_workers_per_gather": map[string]any{
				"description": "Sets the maximum number of workers that can be started by a single Gather or Gather Merge node",
				"maximum":     96,
				"type":        "integer",
				"title":       "max_parallel_workers_per_gather",
				"minimum":     0,
			},
			"bgwriter_lru_multiplier": map[string]any{
				"description": "The average recent need for new buffers is multiplied by bgwriter_lru_multiplier to arrive at an estimate of the number that will be needed during the next round, (up to bgwriter_lru_maxpages). 1.0 represents a “just in time” policy of writing exactly the number of buffers predicted to be needed. Larger values provide some cushion against spikes in demand, while smaller values intentionally leave writes to be done by server processes. The default is 2.0.",
				"maximum":     10,
				"type":        "number",
				"title":       "bgwriter_lru_multiplier",
				"minimum":     0,
				"example":     2.0,
			},
			"pg_partman_bgw.interval": map[string]any{
				"description": "Sets the time interval to run pg_partman's scheduled tasks",
				"maximum":     604800,
				"type":        "integer",
				"title":       "pg_partman_bgw.interval",
				"minimum":     3600,
				"example":     3600,
			},
			"autovacuum_naptime": map[string]any{
				"description": "Specifies the minimum delay between autovacuum runs on any given database. The delay is measured in seconds, and the default is one minute",
				"maximum":     86400,
				"type":        "integer",
				"title":       "autovacuum_naptime",
				"minimum":     1,
			},
			"log_line_prefix": map[string]any{
				"description": "Choose from one of the available log-formats. These can support popular log analyzers like pgbadger, pganalyze etc.",
				"enum": []string{
					"'pid=%p,user=%u,db=%d,app=%a,client=%h '",
					"'%t [%p]: [%l-1] user=%u,db=%d,app=%a,client=%h '",
					"'%m [%p] %q[user=%u,db=%d,app=%a] '",
				},
				"type":  "string",
				"title": "log_line_prefix",
			},
			"log_temp_files": map[string]any{
				"description": "Log statements for each temporary file created larger than this number of kilobytes, -1 disables",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "log_temp_files",
				"minimum":     -1,
			},
			"max_locks_per_transaction": map[string]any{
				"description": "PostgreSQL maximum locks per transaction",
				"maximum":     6400,
				"type":        "integer",
				"title":       "max_locks_per_transaction",
				"minimum":     64,
			},
			"autovacuum_vacuum_scale_factor": map[string]any{
				"description": "Specifies a fraction of the table size to add to autovacuum_vacuum_threshold when deciding whether to trigger a VACUUM. The default is 0.2 (20% of table size)",
				"maximum":     1.0,
				"type":        "number",
				"title":       "autovacuum_vacuum_scale_factor",
				"minimum":     0.0,
			},
			"wal_writer_delay": map[string]any{
				"description": "WAL flush interval in milliseconds. Note that setting this value to lower than the default 200ms may negatively impact performance",
				"maximum":     200,
				"type":        "integer",
				"title":       "wal_writer_delay",
				"minimum":     10,
				"example":     50,
			},
			"track_commit_timestamp": map[string]any{
				"description": "Record commit time of transactions.",
				"enum": []string{
					"off",
					"on",
				},
				"type":    "string",
				"title":   "track_commit_timestamp",
				"example": "off",
			},
			"track_functions": map[string]any{
				"description": "Enables tracking of function call counts and time used.",
				"enum": []string{
					"all",
					"pl",
					"none",
				},
				"type":  "string",
				"title": "track_functions",
			},
			"wal_sender_timeout": map[string]any{
				"description": "Terminate replication connections that are inactive for longer than this amount of time, in milliseconds. Setting this value to zero disables the timeout.",
				"anyOf": []map[string]int{
					{
						"maximum": 0,
						"minimum": 0,
					},
					{
						"maximum": 10800000,
						"minimum": 5000,
					},
				},
				"type":       "integer",
				"user_error": "Must be either 0 or between 5000 and 10800000.",
				"title":      "wal_sender_timeout",
				"example":    60000,
			},
			"autovacuum_vacuum_cost_delay": map[string]any{
				"description": "Specifies the cost delay value that will be used in automatic VACUUM operations. If -1 is specified, the regular vacuum_cost_delay value will be used. The default value is 20 milliseconds",
				"maximum":     100,
				"type":        "integer",
				"title":       "autovacuum_vacuum_cost_delay",
				"minimum":     -1,
			},
			"max_stack_depth": map[string]any{
				"description": "Maximum depth of the stack in bytes",
				"maximum":     6291456,
				"type":        "integer",
				"title":       "max_stack_depth",
				"minimum":     2097152,
			},
			"max_parallel_workers": map[string]any{
				"description": "Sets the maximum number of workers that the system can support for parallel queries",
				"maximum":     96,
				"type":        "integer",
				"title":       "max_parallel_workers",
				"minimum":     0,
			},
			"pg_partman_bgw.role": map[string]any{
				"description": "Controls which role to use for pg_partman's scheduled background tasks.",
				"type":        "string",
				"user_error":  "Must consist of alpha-numeric characters, dots, underscores or dashes, may not start with dash or dot, max 64 characters",
				"title":       "pg_partman_bgw.role",
				"maxLength":   64,
				"example":     "myrolename",
				"pattern":     "^[_A-Za-z0-9][-._A-Za-z0-9]{0,63}$",
			},
			"max_wal_senders": map[string]any{
				"description": "PostgreSQL maximum WAL senders",
				"maximum":     64,
				"type":        "integer",
				"title":       "max_wal_senders",
				"minimum":     20,
			},
			"max_logical_replication_workers": map[string]any{
				"description": "PostgreSQL maximum logical replication workers (taken from the pool of max_parallel_workers)",
				"maximum":     64,
				"type":        "integer",
				"title":       "max_logical_replication_workers",
				"minimum":     4,
			},
			"autovacuum_analyze_scale_factor": map[string]any{
				"description": "Specifies a fraction of the table size to add to autovacuum_analyze_threshold when deciding whether to trigger an ANALYZE. The default is 0.2 (20% of table size)",
				"maximum":     1.0,
				"type":        "number",
				"title":       "autovacuum_analyze_scale_factor",
				"minimum":     0.0,
			},
			"max_prepared_transactions": map[string]any{
				"description": "PostgreSQL maximum prepared transactions",
				"maximum":     10000,
				"type":        "integer",
				"title":       "max_prepared_transactions",
				"minimum":     0,
			},
			"autovacuum_analyze_threshold": map[string]any{
				"description": "Specifies the minimum number of inserted, updated or deleted tuples needed to trigger an  ANALYZE in any one table. The default is 50 tuples.",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "autovacuum_analyze_threshold",
				"minimum":     0,
			},
			"max_worker_processes": map[string]any{
				"description": "Sets the maximum number of background processes that the system can support",
				"maximum":     96,
				"type":        "integer",
				"title":       "max_worker_processes",
				"minimum":     8,
			},
			"pg_stat_statements.track": map[string]any{
				"description": "Controls which statements are counted. Specify top to track top-level statements (those issued directly by clients), all to also track nested statements (such as statements invoked within functions), or none to disable statement statistics collection. The default value is top.",
				"enum": []string{
					"all",
					"top",
					"none",
				},
				"type": []string{
					"string",
				},
				"title": "pg_stat_statements.track",
			},
			"temp_file_limit": map[string]any{
				"description": "PostgreSQL temporary file limit in KiB, -1 for unlimited",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "temp_file_limit",
				"minimum":     -1,
				"example":     5000000,
			},
			"bgwriter_lru_maxpages": map[string]any{
				"description": "In each round, no more than this many buffers will be written by the background writer. Setting this to zero disables background writing. Default is 100.",
				"maximum":     1073741823,
				"type":        "integer",
				"title":       "bgwriter_lru_maxpages",
				"minimum":     0,
				"example":     100,
			},
			"log_error_verbosity": map[string]any{
				"description": "Controls the amount of detail written in the server log for each message that is logged.",
				"enum": []string{
					"TERSE",
					"DEFAULT",
					"VERBOSE",
				},
				"type":  "string",
				"title": "log_error_verbosity",
			},
			"autovacuum_freeze_max_age": map[string]any{
				"description": "Specifies the maximum age (in transactions) that a table's pg_class.relfrozenxid field can attain before a VACUUM operation is forced to prevent transaction ID wraparound within the table. Note that the system will launch autovacuum processes to prevent wraparound even when autovacuum is otherwise disabled. This parameter will cause the server to be restarted.",
				"maximum":     1500000000,
				"type":        "integer",
				"title":       "autovacuum_freeze_max_age",
				"minimum":     200000000,
				"example":     200000000,
			},
			"log_min_duration_statement": map[string]any{
				"description": "Log statements that take more than this number of milliseconds to run, -1 disables",
				"maximum":     86400000,
				"type":        "integer",
				"title":       "log_min_duration_statement",
				"minimum":     -1,
			},
			"max_standby_streaming_delay": map[string]any{
				"description": "Max standby streaming delay in milliseconds",
				"maximum":     43200000,
				"type":        "integer",
				"title":       "max_standby_streaming_delay",
				"minimum":     1,
			},
			"jit": map[string]any{
				"description": "Controls system-wide use of Just-in-Time Compilation (JIT).",
				"type":        "boolean",
				"title":       "jit",
				"example":     true,
			},
			"max_standby_archive_delay": map[string]any{
				"description": "Max standby archive delay in milliseconds",
				"maximum":     43200000,
				"type":        "integer",
				"title":       "max_standby_archive_delay",
				"minimum":     1,
			},
			"max_slot_wal_keep_size": map[string]any{
				"description": "PostgreSQL maximum WAL size (MB) reserved for replication slots. Default is -1 (unlimited). wal_keep_size minimum WAL size setting takes precedence over this.",
				"maximum":     2147483647,
				"type":        "integer",
				"title":       "max_slot_wal_keep_size",
				"minimum":     -1,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "postgresql.conf configuration values",
	},
	Pglookout: &exoscalesdk.GetDBAASSettingsPGResponseSettingsPglookout{
		Properties: map[string]any{
			"max_failover_replication_time_lag": map[string]any{
				"description": "Number of seconds of master unavailability before triggering database failover to standby",
				"default":     60,
				"maximum":     9223372036854775807,
				"type":        "integer",
				"title":       "max_failover_replication_time_lag",
				"minimum":     10,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "PGLookout settings",
	},
	Pgbouncer: &exoscalesdk.GetDBAASSettingsPGResponseSettingsPgbouncer{
		Properties: map[string]any{
			"min_pool_size": map[string]any{
				"maximum": 10000,
				"type":    "integer",
				"title":   "Add more server connections to pool if below this number. Improves behavior when usual load comes suddenly back after period of total inactivity. The value is effectively capped at the pool size.",
				"minimum": 0,
				"example": 0,
			},
			"ignore_startup_parameters": map[string]any{
				"type":  "array",
				"title": "List of parameters to ignore when given in startup packet",
				"example": []string{
					"extra_float_digits",
					"search_path",
				},
				"items": map[string]any{
					"enum": []string{
						"extra_float_digits",
						"search_path",
					},
					"type":  "string",
					"title": "Enum of parameters to ignore when given in startup packet",
				},
				"maxItems": 32,
			},
			"server_lifetime": map[string]any{
				"maximum": 86400,
				"type":    "integer",
				"title":   "The pooler will close an unused server connection that has been connected longer than this. [seconds]",
				"minimum": 60,
				"example": 3600,
			},
			"autodb_pool_mode": map[string]any{
				"enum": []string{
					"session",
					"transaction",
					"statement",
				},
				"type":    "string",
				"title":   "PGBouncer pool mode",
				"example": "session",
			},
			"server_idle_timeout": map[string]any{
				"maximum": 86400,
				"type":    "integer",
				"title":   "If a server connection has been idle more than this many seconds it will be dropped. If 0 then timeout is disabled. [seconds]",
				"minimum": 0,
				"example": 600,
			},
			"autodb_max_db_connections": map[string]any{
				"maximum": 2147483647,
				"type":    "integer",
				"title":   "Do not allow more than this many server connections per database (regardless of user). Setting it to 0 means unlimited.",
				"minimum": 0,
				"example": 0,
			},
			"server_reset_query_always": map[string]any{
				"type":    "boolean",
				"title":   "Run server_reset_query (DISCARD ALL) in all pooling modes",
				"example": false,
			},
			"autodb_pool_size": map[string]any{
				"maximum": 10000,
				"type":    "integer",
				"title":   "If non-zero then create automatically a pool of that size per user when a pool doesn't exist.",
				"minimum": 0,
				"example": 0,
			},
			"autodb_idle_timeout": map[string]any{
				"maximum": 86400,
				"type":    "integer",
				"title":   "If the automatically created database pools have been unused this many seconds, they are freed. If 0 then timeout is disabled. [seconds]",
				"minimum": 0,
				"example": 3600,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "PGBouncer connection pooling settings",
	},
	Timescaledb: &exoscalesdk.GetDBAASSettingsPGResponseSettingsTimescaledb{
		Properties: map[string]any{
			"max_background_workers": map[string]any{
				"description": "The number of background workers for timescaledb operations. You should configure this setting to the sum of your number of databases and the total number of concurrent background workers you want running at any given point in time.",
				"maximum":     4096,
				"type":        "integer",
				"title":       "timescaledb.max_background_workers",
				"minimum":     1,
				"example":     8,
			},
		},
		AdditionalProperties: ptr.To[bool](false),
		Type:                 "object",
		Title:                "TimescaleDB extension configuration values",
	},
}
