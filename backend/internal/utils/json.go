package utils

import (
	"encoding/json"
	"log"

	"gorm.io/datatypes"
)

// ToJSON converts a map or struct to datatypes.JSON for GORM
func ToJSON(data interface{}) (datatypes.JSON, error) {
	if data == nil {
		return datatypes.JSON("{}"), nil
	}

	bytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return datatypes.JSON(bytes), nil
}

// FromJSON parses datatypes.JSON into a map
func FromJSON(data datatypes.JSON) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// DeepMergeJSON merges overrides into base JSON. For each key in overrides:
// if both values are maps, merge recursively; otherwise override wins.
// Returns the original base if overrides is nil/empty or parsing fails.
func DeepMergeJSON(base, overrides datatypes.JSON) datatypes.JSON {
	if len(overrides) == 0 {
		return base
	}

	var baseMap map[string]interface{}
	var overridesMap map[string]interface{}

	if err := json.Unmarshal(base, &baseMap); err != nil {
		log.Printf("DeepMergeJSON: failed to unmarshal base: %v", err)
		baseMap = make(map[string]interface{})
	}
	if err := json.Unmarshal(overrides, &overridesMap); err != nil {
		return base
	}

	merged := deepMergeMaps(baseMap, overridesMap)

	result, err := json.Marshal(merged)
	if err != nil {
		return base
	}
	return datatypes.JSON(result)
}

func deepMergeMaps(base, overrides map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overrides {
		if baseMap, ok := result[k].(map[string]interface{}); ok {
			if overrideMap, ok2 := v.(map[string]interface{}); ok2 {
				result[k] = deepMergeMaps(baseMap, overrideMap)
				continue
			}
		}
		result[k] = v
	}
	return result
}
