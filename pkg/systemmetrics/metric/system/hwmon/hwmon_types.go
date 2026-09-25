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
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/go-structform"
)

var baseDir = "/sys/class/hwmon"

// MonData is a simple wrapper type for the map returned by ReportSensors
type MonData map[string]SensorMetrics

// ErrNoMetric specifies that no metrics were found for a given Sensor.
// This is meant to be a soft error if needed, as the slapdash nature of hwmon sysfs files
// means that we can see *_label files with no corresponding metrics, and so on
var ErrNoMetric = errors.New("no Metrics exist in this device")

// Device represents a single sensor chip, usually exposed as /sys/class/hwmon/hwmon*
type Device struct {
	// Name is the specified hwmon label for the directory
	Name string
	// AbsPath is the absolute path to the monitoring directory, usually linked as /sys/class/hwmon/hwmon*
	AbsPath string
	//Sensors are the individual metrics connected to a device
	Sensors []Sensor
}

// SensorType is for the string prefix of the sensor files
type SensorType struct {
	fileKey string
	units   string
}

// TempSensor is the string prefix for a temp sensor
var TempSensor = SensorType{fileKey: "temp", units: "celsius"}

// VoltSensor is the prefix for voltage sensors
var VoltSensor = SensorType{fileKey: "in", units: "millivolts"}

// FanSensor is the prefix for fan sensors
var FanSensor = SensorType{fileKey: "fan", units: "rpm"}

// Sensor is used to track a single hwmon chip metric
type Sensor struct {
	DevType SensorType
	// SensorNum is the numerical ID of the sensor, i.e temp7_*
	SensorNum int64
}

// SensorMetrics reports the actual metrics in a sensor
// This is meant to be generic for all possible sensor types.
type SensorMetrics struct {
	//Generic Fields
	Label string `struct:"label,omitempty"`

	// This field gets inserted into the map, before individual values.
	sensorType SensorType

	Critical opt.Uint `struct:"critical,omitempty"`
	Max      opt.Uint `struct:"max,omitempty"`
	Lowest   opt.Uint `struct:"lowest,omitempty"`
	Average  opt.Uint `struct:"average,omitempty"`

	// The input value of the metric. The key is overridden and set to the value of sensorType by Fold()
	Value opt.Uint `struct:"value,omitempty"`
}

// Fold implements the Folder interface for structform
// This is entirely so we can carry around a relatively simple struct that transforms into the more heavily nested event that's the standard for beats events.
func (sm *SensorMetrics) Fold(v structform.ExtVisitor) error {
	val := reflect.ValueOf(sm).Elem()
	types := reflect.TypeFor[SensorMetrics]()

	err := v.OnObjectStart(val.NumField(), structform.AnyType)
	if err != nil {
		return fmt.Errorf("error starting object in Fold: %w", err)
	}

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).CanInterface() {

			// Fetch the struct tags
			iface := val.Field(i).Interface()
			skey, tagExists := types.Field(i).Tag.Lookup("struct")
			if !tagExists {
				skey = types.Field(i).Name
			} else {
				skey = strings.Split(skey, ",")[0]
			}

			if skey == "value" {
				skey = sm.sensorType.fileKey
			}

			// Cast to an opt type, then create a nested dict
			castUint, ok := iface.(opt.Uint)
			if ok && castUint.Exists() {
				err := v.OnKey(skey)
				if err != nil {
					return fmt.Errorf("error in OnKey for %s: %w", skey, err)
				}
				// This "inserts" the unit of the metric into the dict as an extra key
				err = v.OnUint64Object(map[string]uint64{sm.sensorType.units: castUint.ValueOr(0)})
				if err != nil {
					return fmt.Errorf("error in OnKey for %s: %w", skey, err)
				}

			}
		}

	}
	err = v.OnObjectFinished()
	if err != nil {
		return fmt.Errorf("error in OnObjectFinished for %s: %w", sm.Label, err)
	}

	return nil
}
