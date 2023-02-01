package mysqlcontroller

import (
	"context"
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"k8s.io/apimachinery/pkg/runtime"
)

type fakeSettingsFetcher struct{}

func (fakeSettingsFetcher) GetDbaasSettingsMysqlWithResponse(ctx context.Context, reqEditors ...oapi.RequestEditorFn) (*oapi.GetDbaasSettingsMysqlResponse, error) {
	return &oapi.GetDbaasSettingsMysqlResponse{
		Body: rawResponse,
	}, nil
}

func mustToRawExt(t *testing.T, set map[string]interface{}) runtime.RawExtension {
	res, err := mapper.ToRawExtension(&set)
	require.NoError(t, err, "failed to parse input setting")
	return res
}

func TestDefaultSettings(t *testing.T) {
	found := exoscalev1.MySQLParameters{
		Maintenance: exoscalev1.MaintenanceSpec{},
		Zone:        "gva-2",
		DBaaSParameters: exoscalev1.DBaaSParameters{
			TerminationProtection: false,
			Size: exoscalev1.SizeSpec{
				Plan: "startup-4",
			},
		},
		MySQLSettings: mustToRawExt(t, map[string]interface{}{
			"net_write_timeout": 24,
		}),
	}

	withDefaults, err := setSettingsDefaults(context.Background(), fakeSettingsFetcher{}, &found)
	require.NoError(t, err, "failed to set defaults")
	setingsWithDefaults, err := mapper.ToMap(withDefaults.MySQLSettings)
	require.NoError(t, err, "failed to parse set defaults")
	assert.EqualValues(t, 24, setingsWithDefaults["net_write_timeout"])
	assert.Len(t, setingsWithDefaults, 1)
}

var rawResponse = []byte(`{"settings":{"mysql":{"properties":{"net_write_timeout":{"description":"The number of seconds to wait for a block to be written to a connection before aborting the write.","maximum":3600,"type":"integer","title":"net_write_timeout","minimum":1,"example":30},"internal_tmp_mem_storage_engine":{"description":"The storage engine for in-memory internal temporary tables.","enum":["TempTable","MEMORY"],"type":"string","title":"internal_tmp_mem_storage_engine","example":"TempTable"},"sql_mode":{"description":"Global SQL mode. Set to empty to use MySQL server defaults. When creating a new service and not setting this field Aiven default SQL mode (strict, SQL standard compliant) will be assigned.","type":"string","user_error":"Must be uppercase alphabetic characters, underscores and commas","title":"sql_mode","maxLength":1024,"example":"ANSI,TRADITIONAL","pattern":"^[A-Z_]*(,[A-Z_]+)*$"},"information_schema_stats_expiry":{"description":"The time, in seconds, before cached statistics expire","maximum":31536000,"type":"integer","title":"information_schema_stats_expiry","minimum":900,"example":86400},"sort_buffer_size":{"description":"Sort buffer size in bytes for ORDER BY optimization. Default is 262144 (256K)","maximum":1073741824,"type":"integer","title":"sort_buffer_size","minimum":32768,"example":262144},"innodb_thread_concurrency":{"description":"Defines the maximum number of threads permitted inside of InnoDB. Default is 0 (infinite concurrency - no limit)","maximum":1000,"type":"integer","title":"innodb_thread_concurrency","minimum":0,"example":10},"innodb_write_io_threads":{"description":"The number of I/O threads for write operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service.","maximum":64,"type":"integer","title":"innodb_write_io_threads","minimum":1,"example":10},"innodb_ft_min_token_size":{"description":"Minimum length of words that are stored in an InnoDB FULLTEXT index. Changing this parameter will lead to a restart of the MySQL service.","maximum":16,"type":"integer","title":"innodb_ft_min_token_size","minimum":0,"example":3},"innodb_change_buffer_max_size":{"description":"Maximum size for the InnoDB change buffer, as a percentage of the total size of the buffer pool. Default is 25","maximum":50,"type":"integer","title":"innodb_change_buffer_max_size","minimum":0,"example":30},"innodb_flush_neighbors":{"description":"Specifies whether flushing a page from the InnoDB buffer pool also flushes other dirty pages in the same extent (default is 1): 0 - dirty pages in the same extent are not flushed,  1 - flush contiguous dirty pages in the same extent,  2 - flush dirty pages in the same extent","maximum":2,"type":"integer","title":"innodb_flush_neighbors","minimum":0,"example":0},"tmp_table_size":{"description":"Limits the size of internal in-memory tables. Also set max_heap_table_size. Default is 16777216 (16M)","maximum":1073741824,"type":"integer","title":"tmp_table_size","minimum":1048576,"example":16777216},"slow_query_log":{"description":"Slow query log enables capturing of slow queries. Setting slow_query_log to false also truncates the mysql.slow_log table. Default is off","type":"boolean","title":"slow_query_log","example":true},"connect_timeout":{"description":"The number of seconds that the mysqld server waits for a connect packet before responding with Bad handshake","maximum":3600,"type":"integer","title":"connect_timeout","minimum":2,"example":10},"net_read_timeout":{"description":"The number of seconds to wait for more data from a connection before aborting the read.","maximum":3600,"type":"integer","title":"net_read_timeout","minimum":1,"example":30},"innodb_lock_wait_timeout":{"description":"The length of time in seconds an InnoDB transaction waits for a row lock before giving up.","maximum":3600,"type":"integer","title":"innodb_lock_wait_timeout","minimum":1,"example":50},"wait_timeout":{"description":"The number of seconds the server waits for activity on a noninteractive connection before closing it.","maximum":2147483,"type":"integer","title":"wait_timeout","minimum":1,"example":28800},"innodb_rollback_on_timeout":{"description":"When enabled a transaction timeout causes InnoDB to abort and roll back the entire transaction. Changing this parameter will lead to a restart of the MySQL service.","type":"boolean","title":"innodb_rollback_on_timeout","example":true},"group_concat_max_len":{"description":"The maximum permitted result length in bytes for the GROUP_CONCAT() function.","maximum":18446744073709551615,"type":"integer","title":"group_concat_max_len","minimum":4,"example":1024},"net_buffer_length":{"description":"Start sizes of connection buffer and result buffer. Default is 16384 (16K). Changing this parameter will lead to a restart of the MySQL service.","maximum":1048576,"type":"integer","title":"net_buffer_length","minimum":1024,"example":16384},"innodb_print_all_deadlocks":{"description":"When enabled, information about all deadlocks in InnoDB user transactions is recorded in the error log. Disabled by default.","type":"boolean","title":"innodb_print_all_deadlocks","example":true},"innodb_online_alter_log_max_size":{"description":"The upper limit in bytes on the size of the temporary log files used during online DDL operations for InnoDB tables.","maximum":1099511627776,"type":"integer","title":"innodb_online_alter_log_max_size","minimum":65536,"example":134217728},"interactive_timeout":{"description":"The number of seconds the server waits for activity on an interactive connection before closing it.","maximum":604800,"type":"integer","title":"interactive_timeout","minimum":30,"example":3600},"innodb_log_buffer_size":{"description":"The size in bytes of the buffer that InnoDB uses to write to the log files on disk.","maximum":4294967295,"type":"integer","title":"innodb_log_buffer_size","minimum":1048576,"example":16777216},"max_allowed_packet":{"description":"Size of the largest message in bytes that can be received by the server. Default is 67108864 (64M)","maximum":1073741824,"type":"integer","title":"max_allowed_packet","minimum":102400,"example":67108864},"max_heap_table_size":{"description":"Limits the size of internal in-memory tables. Also set tmp_table_size. Default is 16777216 (16M)","maximum":1073741824,"type":"integer","title":"max_heap_table_size","minimum":1048576,"example":16777216},"innodb_ft_server_stopword_table":{"description":"This option is used to specify your own InnoDB FULLTEXT index stopword list for all InnoDB tables.","type":["null","string"],"title":"innodb_ft_server_stopword_table","maxLength":1024,"example":"db_name/table_name","pattern":"^.+/.+$"},"innodb_read_io_threads":{"description":"The number of I/O threads for read operations in InnoDB. Default is 4. Changing this parameter will lead to a restart of the MySQL service.","maximum":64,"type":"integer","title":"innodb_read_io_threads","minimum":1,"example":10},"sql_require_primary_key":{"description":"Require primary key to be defined for new tables or old tables modified with ALTER TABLE and fail if missing. It is recommended to always have primary keys because various functionality may break if any large table is missing them.","type":"boolean","title":"sql_require_primary_key","example":true},"default_time_zone":{"description":"Default server time zone as an offset from UTC (from -12:00 to +12:00), a time zone name, or 'SYSTEM' to use the MySQL server default.","type":"string","minLength":2,"title":"default_time_zone","maxLength":100,"example":"+03:00"},"long_query_time":{"description":"The slow_query_logs work as SQL statements that take more than long_query_time seconds to execute. Default is 10s","maximum":3600,"type":"number","title":"long_query_time","minimum":0.0,"example":10}},"additionalProperties":false,"type":"object","title":"mysql.conf configuration values"}}}`)
