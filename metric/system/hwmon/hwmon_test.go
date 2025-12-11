// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux

package hwmon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

func TestExamples(t *testing.T) {
	poweredge := "./testdata/PoweredgeR720"
	thinkpad := "./testdata/ThinkpadX250"

	// test Poweredge values
	results, err := DetectHwmon(resolve.NewTestResolver(poweredge))
	assert.NoError(t, err)
	sensors := results[0].Sensors
	assert.Len(t, sensors, 7)
	sensorMetrics, err := ReportSensors(results[0])
	assert.NoError(t, err)
	assert.Len(t, sensorMetrics, 7)
	assert.Equal(t, sensorMetrics["core_4"].Value.ValueOr(0), uint64(52))
	assert.Equal(t, sensorMetrics["core_3"].Label, "Core 3")
	assert.Equal(t, sensorMetrics["core_3"].Max.ValueOr(0), uint64(81))

	// Test Thinkpad
	resultsx250, err := DetectHwmon(resolve.NewTestResolver(thinkpad))
	assert.NoError(t, err)
	// thinkpad thermal sensors
	sensorsTP := resultsx250[1].Sensors
	assert.Len(t, resultsx250, 2)
	assert.Len(t, sensorsTP, 9)
	sensorMetricsTP, err := ReportSensors(resultsx250[1])
	assert.NoError(t, err)
	assert.Equal(t, uint64(2200), sensorMetricsTP["fan_1"].Value.ValueOr(0))
	assert.Equal(t, uint64(76), sensorMetricsTP["temp_6"].Value.ValueOr(0))
	assert.Equal(t, uint64(0), sensorMetricsTP["temp_7"].Value.ValueOr(0))
	//thinkpad battery sensor
	sensorMetricsBAT, err := ReportSensors(resultsx250[0])
	assert.NoError(t, err)
	assert.Equal(t, uint64(11943), sensorMetricsBAT["in_0"].Value.ValueOr(0))

}

func TestFetch(t *testing.T) {
	// This is meant to test how this library would be used by a metricset.
	/*
			Example output:
			Sensor coretemp: {
		          "core_0": {
		            "critical": {
		              "celsius": 91
		            },
		            "max": {
		              "celsius": 81
		            },
		            "temp": {
		              "celsius": 49
		            }
		          "core_3": {
		            "critical": {
		              "celsius": 91
		            },
		            "max": {
		              "celsius": 81
		            },
		            "temp": {
		              "celsius": 52
		            }
		          },
		          "package_id_0": {
		            "critical": {
		              "celsius": 91
		            },
		            "max": {
		              "celsius": 81
		            },
		            "temp": {
		              "celsius": 53
		            }
		          }
		        }
	*/

	_, err := os.Stat(filepath.Join(baseDir, "hwmon0"))
	if os.IsNotExist(err) {
		t.Logf("No hwerr devices on system, skipping")
		return
	}

	// This would be called in New() and not Fetch(), as the results are not expected to change.
	results, err := DetectHwmon(resolve.NewTestResolver(""))
	assert.NoError(t, err)

	for _, device := range results {
		//Each device should be sent as it's own event, as they represent metrics from different places.
		sensors, err := ReportSensors(device)
		if err != nil {
			t.Fatalf("error reading sensors: %s", err)
		}
		to := mapstr.M{}

		err = typeconv.Convert(&to, sensors)
		if err != nil {
			t.Fatalf("error converting sensor data: %s", err)
		}

		t.Logf("Sensor %s: %s", device.Name, to.StringToPrint())
	}
}
