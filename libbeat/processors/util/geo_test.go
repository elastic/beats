package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/libbeat/common"
)

// parseGeoConfig converts the map into a GeoConfig.
// Going through go-ucfg we test the config to struct transform / validation.
func parseConfig(t *testing.T, configMap map[string]interface{}) GeoConfig {
	config, err := common.NewConfigFrom(configMap)
	require.NoError(t, err)

	geoConfig := GeoConfig{}
	err = config.Unpack(&geoConfig)
	require.NoError(t, err)

	return geoConfig
}

func TestConfigGeoEnabled(t *testing.T) {
	config := map[string]interface{}{
		"name":             "yerevan-am",
		"location":         "40.177200, 44.503490",
		"continent_name":   "Asia",
		"country_iso_code": "AM",
		"region_name":      "Erevan",
		"region_iso_code":  "AM-ER",
		"city_name":        "Yerevan",
	}

	geoMap, err := GeoConfigToMap(parseConfig(t, config))
	require.NoError(t, err)

	for configKey, configValue := range config {
		t.Run(fmt.Sprintf("Check of %s", configKey), func(t *testing.T) {
			v, ok := geoMap[configKey]
			assert.True(t, ok, "key has entry")
			assert.Equal(t, configValue, v)
		})
	}
}

func TestPartialGeo(t *testing.T) {
	config := map[string]interface{}{
		"name":      "yerevan-am",
		"city_name": "  ",
	}

	geoMap, err := GeoConfigToMap(parseConfig(t, config))
	require.NoError(t, err)

	assert.Equal(t, "yerevan-am", geoMap["name"])

	missing := []string{"continent_name", "country_name", "country_iso_code", "region_name", "region_iso_code", "city_name"}

	for _, k := range missing {
		_, exists := geoMap[k]
		assert.False(t, exists, "key should %s should not exist", k)
	}
}

func TestGeoLocationValidation(t *testing.T) {
	locations := []struct {
		str   string
		valid bool
	}{
		{"40.177200, 44.503490", true},
		{"-40.177200, -44.503490", true},
		{"garbage", false},
		{"9999999999", false},
	}

	for _, location := range locations {
		t.Run(fmt.Sprintf("Location %s validation should be %t", location.str, location.valid), func(t *testing.T) {

			geoConfig := parseConfig(t, common.MapStr{"location": location.str})
			geoMap, err := GeoConfigToMap(geoConfig)

			if location.valid {
				require.NoError(t, err)
				require.Equal(t, location.str, geoMap["location"])
			} else {
				require.Error(t, err)
			}
		})
	}
}
