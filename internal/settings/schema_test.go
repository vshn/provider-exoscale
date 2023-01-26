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
          "type": ["string", "null"]
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
