package util

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/elastic/beats/libbeat/common"
)

// GeoConfig contains geo configuration data.
type GeoConfig struct {
	Name           string `config:"name"`
	Location       string `config:"location"`
	ContinentName  string `config:"continent_name"`
	CountryISOCode string `config:"country_iso_code"`
	RegionName     string `config:"region_name"`
	RegionISOCode  string `config:"region_iso_code"`
	CityName       string `config:"city_name"`
}

func GeoConfigToMap(config GeoConfig) (common.MapStr, error) {
	if len(config.Location) > 0 {
		// Regexp matching a number with an optional decimal component
		// Valid numbers: '123', '123.23', etc.
		latOrLon := `\-?\d+(\.\d+)?`

		// Regexp matching a pair of lat lon coordinates.
		// e.g. 40.123, -92.929
		locRegexp := `^\s*` + // anchor to start of string with optional whitespace
			latOrLon + // match the latitude
			`\s*\,\s*` + // match the separator. optional surrounding whitespace
			latOrLon + // match the longitude
			`\s*$` //optional whitespace then end anchor

		if m, _ := regexp.MatchString(locRegexp, config.Location); !m {
			return nil, errors.New(fmt.Sprintf("Invalid lat,lon  string for add_observer_metadata: %s", config.Location))
		}
	}

	geoFields := common.MapStr{
		"name":             config.Name,
		"location":         config.Location,
		"continent_name":   config.ContinentName,
		"country_iso_code": config.CountryISOCode,
		"region_name":      config.RegionName,
		"region_iso_code":  config.RegionISOCode,
		"city_name":        config.CityName,
	}
	// Delete any empty values
	blankStringMatch := regexp.MustCompile(`^\s*$`)
	for k, v := range geoFields {
		vStr := v.(string)
		if blankStringMatch.MatchString(vStr) {
			delete(geoFields, k)
		}
	}

	return geoFields, nil
}
