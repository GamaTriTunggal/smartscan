package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/datatypes"
)

func toJSON(t *testing.T, v interface{}) datatypes.JSON {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return datatypes.JSON(b)
}

func parseJSON(t *testing.T, data datatypes.JSON) map[string]interface{} {
	t.Helper()
	var result map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func TestDeepMergeJSON_NilOverrides(t *testing.T) {
	base := datatypes.JSON(`{"header":{"bg_color":"#3b82f6"},"styling":{"text_color":"#1f2937"}}`)

	// nil overrides
	result := DeepMergeJSON(base, nil)
	assert.JSONEq(t, string(base), string(result))

	// empty overrides
	result = DeepMergeJSON(base, datatypes.JSON{})
	assert.JSONEq(t, string(base), string(result))
}

func TestDeepMergeJSON_EmptyBase(t *testing.T) {
	overrides := datatypes.JSON(`{"header":{"badge_text":"Custom"}}`)

	// nil base
	result := DeepMergeJSON(nil, overrides)
	parsed := parseJSON(t, result)
	assert.Equal(t, "Custom", parsed["header"].(map[string]interface{})["badge_text"])

	// empty object base
	result = DeepMergeJSON(datatypes.JSON(`{}`), overrides)
	parsed = parseJSON(t, result)
	assert.Equal(t, "Custom", parsed["header"].(map[string]interface{})["badge_text"])
}

func TestDeepMergeJSON_ShallowMerge(t *testing.T) {
	base := toJSON(t, map[string]interface{}{
		"key1": "base_val1",
		"key2": "base_val2",
	})
	overrides := toJSON(t, map[string]interface{}{
		"key2": "override_val2",
		"key3": "override_val3",
	})

	result := DeepMergeJSON(base, overrides)
	parsed := parseJSON(t, result)

	assert.Equal(t, "base_val1", parsed["key1"])
	assert.Equal(t, "override_val2", parsed["key2"])
	assert.Equal(t, "override_val3", parsed["key3"])
}

func TestDeepMergeJSON_DeepMerge(t *testing.T) {
	base := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"bg_color":   "#3b82f6",
			"badge_text": "Authentic Product",
			"badge_bg":   "#22c55e",
		},
	})
	overrides := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"badge_text": "Premium Product",
			"new_field":  "new_value",
		},
	})

	result := DeepMergeJSON(base, overrides)
	parsed := parseJSON(t, result)
	header := parsed["header"].(map[string]interface{})

	// Base value preserved
	assert.Equal(t, "#3b82f6", header["bg_color"])
	assert.Equal(t, "#22c55e", header["badge_bg"])
	// Override applied
	assert.Equal(t, "Premium Product", header["badge_text"])
	// New key added
	assert.Equal(t, "new_value", header["new_field"])
}

func TestDeepMergeJSON_OverrideReplacesNonMap(t *testing.T) {
	// When override value is not a map but base value is a map, override wins
	base := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"bg_color": "#3b82f6",
		},
	})
	overrides := toJSON(t, map[string]interface{}{
		"header": "simple_string",
	})

	result := DeepMergeJSON(base, overrides)
	parsed := parseJSON(t, result)

	// Override replaces the entire map with a string
	assert.Equal(t, "simple_string", parsed["header"])
}

func TestDeepMergeJSON_NewKeys(t *testing.T) {
	base := toJSON(t, map[string]interface{}{
		"existing": "value",
	})
	overrides := toJSON(t, map[string]interface{}{
		"new_section": map[string]interface{}{
			"field1": "val1",
			"field2": true,
		},
	})

	result := DeepMergeJSON(base, overrides)
	parsed := parseJSON(t, result)

	assert.Equal(t, "value", parsed["existing"])
	newSection := parsed["new_section"].(map[string]interface{})
	assert.Equal(t, "val1", newSection["field1"])
	assert.Equal(t, true, newSection["field2"])
}

func TestDeepMergeJSON_InvalidBaseJSON(t *testing.T) {
	corruptBase := datatypes.JSON(`not valid json`)
	overrides := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"badge_text": "Fallback",
		},
	})

	// Should still produce result with overrides merged into empty map
	result := DeepMergeJSON(corruptBase, overrides)
	parsed := parseJSON(t, result)
	header := parsed["header"].(map[string]interface{})
	assert.Equal(t, "Fallback", header["badge_text"])
}

func TestDeepMergeJSON_InvalidOverridesJSON(t *testing.T) {
	base := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"bg_color": "#3b82f6",
		},
	})
	corruptOverrides := datatypes.JSON(`not valid json`)

	// Should return original base when overrides can't be parsed
	result := DeepMergeJSON(base, corruptOverrides)
	assert.JSONEq(t, string(base), string(result))
}

func TestDeepMergeJSON_RealTemplateConfig(t *testing.T) {
	// Realistic scenario: template base config + product overrides
	base := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"logo_enabled":    true,
			"bg_color":        "#3b82f6",
			"badge_text":      "Authentic Product",
			"badge_bg_color":  "#22c55e",
			"badge_text_color": "#ffffff",
		},
		"styling": map[string]interface{}{
			"card_bg_color":  "#f3f4f6",
			"field_bg_color": "#ffffff",
			"text_color":     "#1f2937",
			"main_image_size": 96,
		},
		"certifications_section": map[string]interface{}{
			"header_text":      "Certifications",
			"icon_color":       "#10b981",
			"default_expanded": false,
		},
		"warranty_button": map[string]interface{}{
			"text":       "Activate Warranty",
			"bg_color":   "#8b5cf6",
			"text_color": "#ffffff",
		},
		"section_order": []string{
			"images", "videos", "social_accounts", "certifications",
			"website_link", "description", "warranty_button",
		},
	})

	overrides := toJSON(t, map[string]interface{}{
		"header": map[string]interface{}{
			"badge_text":     "Premium Quality",
			"badge_bg_color": "#ff6600",
		},
		"styling": map[string]interface{}{
			"text_color": "#333333",
		},
		"warranty_button": map[string]interface{}{
			"text": "Register Warranty",
		},
		"section_order": []string{
			"certifications", "images", "description",
		},
	})

	result := DeepMergeJSON(base, overrides)
	parsed := parseJSON(t, result)

	// Header: deep merged
	header := parsed["header"].(map[string]interface{})
	assert.Equal(t, true, header["logo_enabled"])        // preserved
	assert.Equal(t, "#3b82f6", header["bg_color"])        // preserved
	assert.Equal(t, "Premium Quality", header["badge_text"]) // overridden
	assert.Equal(t, "#ff6600", header["badge_bg_color"])     // overridden
	assert.Equal(t, "#ffffff", header["badge_text_color"])   // preserved

	// Styling: deep merged
	styling := parsed["styling"].(map[string]interface{})
	assert.Equal(t, "#f3f4f6", styling["card_bg_color"])  // preserved
	assert.Equal(t, "#ffffff", styling["field_bg_color"])  // preserved
	assert.Equal(t, "#333333", styling["text_color"])      // overridden
	assert.Equal(t, float64(96), styling["main_image_size"]) // preserved (JSON numbers are float64)

	// Certifications: untouched
	certs := parsed["certifications_section"].(map[string]interface{})
	assert.Equal(t, "Certifications", certs["header_text"])

	// Warranty button: deep merged
	warranty := parsed["warranty_button"].(map[string]interface{})
	assert.Equal(t, "Register Warranty", warranty["text"]) // overridden
	assert.Equal(t, "#8b5cf6", warranty["bg_color"])       // preserved

	// Section order: replaced entirely (not a map, so no deep merge)
	sectionOrder := parsed["section_order"].([]interface{})
	assert.Len(t, sectionOrder, 3)
	assert.Equal(t, "certifications", sectionOrder[0])
}
