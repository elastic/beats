// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package device_health

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/meraki"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/go-resty/resty/v2"
	sdk "github.com/meraki/dashboard-api-go/v3/sdk"
)

// SensorClient defines the interface for sensor API calls
type SensorClient interface {
	GetOrganizationSensorReadingsHistory(organizationID string, params *sdk.GetOrganizationSensorReadingsHistoryQueryParams) (*sdk.ResponseSensorGetOrganizationSensorReadingsHistory, *resty.Response, error)
}

var _ SensorClient = (*sdk.SensorService)(nil)

// SensorServiceWrapper wraps the SDK SensorService to implement SensorClient interface
type SensorServiceWrapper struct {
	service *sdk.SensorService
}

func (w *SensorServiceWrapper) GetOrganizationSensorReadingsHistory(organizationID string, params *sdk.GetOrganizationSensorReadingsHistoryQueryParams) (*sdk.ResponseSensorGetOrganizationSensorReadingsHistory, *resty.Response, error) {
	return w.service.GetOrganizationSensorReadingsHistory(organizationID, params)
}

// getSensorReadingsHistory fetches sensor readings history for an organization
func getSensorReadingsHistory(client SensorClient, organizationID string, collectionPeriod time.Duration, logger *logp.Logger) ([]sdk.ResponseItemSensorGetOrganizationSensorReadingsHistory, error) {
	var readings []sdk.ResponseItemSensorGetOrganizationSensorReadingsHistory

	params := &sdk.GetOrganizationSensorReadingsHistoryQueryParams{
		Timespan: collectionPeriod.Seconds(),
	}
	setStart := func(s string) { params.StartingAfter = s }

	doRequest := func() (*sdk.ResponseSensorGetOrganizationSensorReadingsHistory, *resty.Response, error) {
		logger.Debugf("calling GetOrganizationSensorReadingsHistory with params: %+v", params)
		return client.GetOrganizationSensorReadingsHistory(organizationID, params)
	}

	onError := func(err error, res *resty.Response) error {
		if res != nil {
			return fmt.Errorf("GetOrganizationSensorReadingsHistory failed; [%d] %s. %w", res.StatusCode(), res.Body(), err)
		}
		return fmt.Errorf("GetOrganizationSensorReadingsHistory failed; %w", err)
	}

	onSuccess := func(val *sdk.ResponseSensorGetOrganizationSensorReadingsHistory) error {
		if val == nil {
			return errors.New("GetOrganizationSensorReadingsHistory returned nil response")
		}

		readings = append(readings, *val...)
		return nil
	}

	err := meraki.NewPaginator(
		setStart,
		doRequest,
		onError,
		onSuccess,
		logger,
	).GetAllPages()

	return readings, err
}

// reportSensorReadings converts sensor readings to events and reports them
func reportSensorReadings(reporter mb.ReporterV2, organizationID string, readings []sdk.ResponseItemSensorGetOrganizationSensorReadingsHistory, devices map[Serial]*Device) {
	metrics := make([]mapstr.M, 0, len(readings))

	for _, reading := range readings {
		// Start with device details if we can find the device by serial
		var metric mapstr.M
		if device, ok := devices[Serial(reading.Serial)]; ok && device != nil && device.details != nil {
			metric = deviceDetailsToMapstr(device.details)
		} else {
			metric = mapstr.M{}
		}

		// Add sensor-specific fields
		addSensorReadingFields(metric, &reading)
		metrics = append(metrics, metric)
	}

	meraki.ReportMetricsForOrganization(reporter, organizationID, metrics)
}

// addSensorReadingFields adds sensor reading fields to the given mapstr.M
func addSensorReadingFields(m mapstr.M, reading *sdk.ResponseItemSensorGetOrganizationSensorReadingsHistory) {
	m["sensor.serial"] = reading.Serial
	m["sensor.metric"] = reading.Metric
	m["@timestamp"] = reading.Ts

	if reading.Network != nil {
		m["sensor.network.id"] = reading.Network.ID
		m["sensor.network.name"] = reading.Network.Name
	}

	if reading.ApparentPower != nil {
		m["sensor.apparent_power.draw"] = reading.ApparentPower.Draw
	}

	if reading.Battery != nil {
		m["sensor.battery.percentage"] = reading.Battery.Percentage
	}

	if reading.Button != nil {
		m["sensor.button.press_type"] = reading.Button.PressType
	}

	if reading.Co2 != nil {
		m["sensor.co2.concentration"] = reading.Co2.Concentration
	}

	if reading.Current != nil {
		m["sensor.current.draw"] = reading.Current.Draw
	}

	if reading.Door != nil {
		m["sensor.door.open"] = reading.Door.Open
	}

	if reading.DownstreamPower != nil {
		m["sensor.downstream_power.enabled"] = reading.DownstreamPower.Enabled
	}

	if reading.Frequency != nil {
		m["sensor.frequency.level"] = reading.Frequency.Level
	}

	if reading.Humidity != nil {
		m["sensor.humidity.relative_percentage"] = reading.Humidity.RelativePercentage
	}

	if reading.IndoorAirQuality != nil {
		m["sensor.indoor_air_quality.score"] = reading.IndoorAirQuality.Score
	}

	if reading.Noise != nil && reading.Noise.Ambient != nil {
		m["sensor.noise.ambient.level"] = reading.Noise.Ambient.Level
	}

	if reading.Pm25 != nil {
		m["sensor.pm25.concentration"] = reading.Pm25.Concentration
	}

	if reading.PowerFactor != nil {
		m["sensor.power_factor.percentage"] = reading.PowerFactor.Percentage
	}

	if reading.RealPower != nil {
		m["sensor.real_power.draw"] = reading.RealPower.Draw
	}

	if reading.RemoteLockoutSwitch != nil {
		m["sensor.remote_lockout_switch.locked"] = reading.RemoteLockoutSwitch.Locked
	}

	if reading.Temperature != nil {
		m["sensor.temperature.celsius"] = reading.Temperature.Celsius
		m["sensor.temperature.fahrenheit"] = reading.Temperature.Fahrenheit
	}

	if reading.Tvoc != nil {
		m["sensor.tvoc.concentration"] = reading.Tvoc.Concentration
	}

	if reading.Voltage != nil {
		m["sensor.voltage.level"] = reading.Voltage.Level
	}

	if reading.Water != nil {
		m["sensor.water.present"] = reading.Water.Present
	}
}
