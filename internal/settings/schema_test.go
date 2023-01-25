package settings

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vshn/provider-exoscale/operator/mapper"
	"k8s.io/apimachinery/pkg/runtime"
)

var exampleSchemas = []byte(`{
  "settings": {
    "simple": {
      "type": "object",
      "properties": {
        "foo": {
          "default": true,
          "type": "boolean"
        },
        "bar": {
          "default": "buzz",
          "type": "string"
        },
        "count": {
          "default": 42,
          "type": "integer"
        },
        "nodefault": {
          "type": "string"
        }
      }
    },
    "nested": {
      "type": "object",
      "properties": {
        "foo": {
          "default": true,
          "type": "boolean"
        },
        "obj": {
          "type": "object",
          "properties": {
            "bar": {
              "default": "buzz",
              "type": "string"
            },
            "obj": {
              "type": "object",
              "properties": {
                "count": {
                  "default": 42,
                  "type": "integer"
                }
              }
            }
          }
        }
      }
    }
  }
}`)

var b = `{"type":"object","title":"Redis settings","properties":{"ssl":{"default":true,"type":"boolean","title":"Require SSL to access Redis"},"lfu_log_factor":{"default":10,"maximum":100,"type":"integer","title":"Counter logarithm factor for volatile-lfu and allkeys-lfu maxmemory-policies","minimum":0},"maxmemory_policy":{"enum":["noeviction","allkeys-lru","volatile-lru","allkeys-random","volatile-random","volatile-ttl","volatile-lfu","allkeys-lfu"],"default":"noeviction","type":"string","title":"Redis maxmemory-policy"},"io_threads":{"maximum":32,"type":"integer","title":"Redis IO thread count","minimum":1,"example":1},"lfu_decay_time":{"default":1,"maximum":120,"type":"integer","title":"LFU maxmemory-policy counter decay time in minutes","minimum":1},"pubsub_client_output_buffer_limit":{"description":"Set output buffer limit for pub / sub clients in MB. The value is the hard limit, the soft limit is 1/4 of the hard limit. When setting the limit, be mindful of the available memory in the selected service plan.","maximum":512,"type":"integer","title":"Pub/sub client output buffer hard limit in MB","minimum":32,"example":64},"notify_keyspace_events":{"default":"","type":"string","title":"Set notify-keyspace-events option","maxLength":32,"pattern":"^[KEg\\$lshzxeA]*$"},"persistence":{"description":"When persistence is 'rdb', Redis does RDB dumps each 10 minutes if any key is changed. Also RDB dumps are done according to backup schedule for backup purposes. When persistence is 'off', no RDB dumps and backups are done, so data can be lost at any moment if service is restarted for any reason, or if service is powered off. Also service can't be forked.","enum":["off","rdb"],"type":"string","title":"Redis persistence"},"timeout":{"default":300,"maximum":31536000,"type":"integer","title":"Redis idle connection timeout in seconds","minimum":0},"acl_channels_default":{"description":"Determines default pub/sub channels' ACL for new users if ACL is not supplied. When this option is not defined, all_channels is assumed to keep backward compatibility. This option doesn't affect Redis configuration acl-pubsub-default.","enum":["allchannels","resetchannels"],"type":"string","title":"Default ACL for pub/sub channels used when Redis user is created"},"number_of_databases":{"description":"Set number of redis databases. Changing this will cause a restart of redis service.","maximum":128,"type":"integer","title":"Number of redis databases","minimum":1,"example":16}}}`

func TestSetDefaultSimple(t *testing.T) {
	schemas, err := ParseSchemas(exampleSchemas)
	require.NoError(t, err, "failed to parse example schema")

	input := runtime.RawExtension{
		Raw: []byte(`{"bar": "blub"}`),
	}
	out, err := schemas.SetDefaults("simple", input)
	require.NoError(t, err, "failed to set defaults")

	outMap, err := mapper.ToMap(out)
	require.NoError(t, err, "failed to set defaults")

	assert.EqualValues(t, true, outMap["foo"])
	assert.EqualValues(t, "blub", outMap["bar"])
	assert.EqualValues(t, 42, outMap["count"])
	_, ok := outMap["nodefault"]
	assert.Falsef(t, ok, "should not set values withou defaults")
}

func TestSetDefaultNested(t *testing.T) {
	schemas, err := ParseSchemas(exampleSchemas)
	require.NoError(t, err, "failed to parse example schema")

	input := runtime.RawExtension{
		Raw: []byte(`{"bar": "blub"}`),
	}
	out, err := schemas.SetDefaults("nested", input)
	require.NoError(t, err, "failed to set defaults")

	outMap, err := mapper.ToMap(out)
	require.NoError(t, err, "failed to set defaults")

	assert.EqualValues(t, true, outMap["foo"])
	assert.EqualValues(t, "blub", outMap["bar"])

	sub1, ok := outMap["obj"]
	require.Truef(t, ok, "should set sub object")
	sub1Map, ok := sub1.(map[string]interface{})
	require.Truef(t, ok, "should set sub object as map")
	assert.EqualValues(t, "buzz", sub1Map["bar"])

	sub2, ok := sub1Map["obj"]
	require.Truef(t, ok, "should set sub-sub object")
	sub2Map, ok := sub2.(map[string]interface{})
	require.Truef(t, ok, "should set sub-sub object as map")
	assert.EqualValues(t, 42, sub2Map["count"])
}
