package settings

import (
	"encoding/json"
	"fmt"

	"github.com/vshn/provider-exoscale/operator/mapper"
	"k8s.io/apimachinery/pkg/runtime"
)

type Schemas interface {
	SetDefaults(schema string, input runtime.RawExtension) (runtime.RawExtension, error)
}

// ParseSchemas takes an object containing a map of json schemas and parses it
func ParseSchemas(raw []byte) (Schemas, error) {
	req := struct {
		Settings schemas
	}{}

	err := json.Unmarshal(raw, &req)
	if err != nil {
		return nil, err
	}
	return req.Settings, nil
}

type schemas map[string]schema

type schema struct {
	Default    interface{}
	Properties schemas
}

// SetDefaults takes a setting for a DBaaS and will set the defaults of the schema with name `name`
func (s schemas) SetDefaults(name string, input runtime.RawExtension) (runtime.RawExtension, error) {
	sc, ok := s[name]
	if !ok {
		return runtime.RawExtension{}, fmt.Errorf("unknown schema: %q", name)
	}

	inMap, err := mapper.ToMap(input)
	if err != nil {
		return runtime.RawExtension{}, fmt.Errorf("failed to parse input: %w", err)
	}

	setDefaults(sc, inMap)

	out, err := mapper.ToRawExtension(&inMap)
	if err != nil {
		return runtime.RawExtension{}, fmt.Errorf("failed to parse defaulted setting: %w", err)
	}
	return out, nil
}

func setDefaults(sc schema, input map[string]interface{}) bool {
	hasSetDefaults := false

	for key, val := range sc.Properties {
		if len(val.Properties) > 0 {
			submap := map[string]interface{}{}

			if _, ok := input[key]; ok {
				submap, ok = input[key].(map[string]interface{})
				if !ok {
					continue
				}
			}

			if setDefaults(val, submap) {
				input[key] = submap
				hasSetDefaults = true
			}
		} else {
			_, ok := input[key]
			if ok {
				continue
			}

			if val.Default != nil {
				input[key] = val.Default
				hasSetDefaults = true
			}
		}
	}
	return hasSetDefaults
}
