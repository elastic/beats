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
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// ReportSensors returns the metrics from all the known sensors.
func ReportSensors(dev Device) (MonData, error) {
	metrics := MonData{}
	for _, sensor := range dev.Sensors {
		data, err := sensor.Fetch(dev.AbsPath)
		if errors.Is(err, ErrNoMetric) {
			continue
		} else if err != nil {
			return nil, fmt.Errorf("error fetching sensor data for %s: %w", sensor.DevType, err)
		}
		// Create the device key from the label, a the values are considerably more intuative.
		labelName := strings.ToLower(strings.ReplaceAll(data.Label, " ", "_"))
		metrics[labelName] = data
	}

	return metrics, nil
}

// DetectHwmon returns a list of hwmon sensors found on the system, if they exist
func DetectHwmon(hostfs resolve.Resolver) ([]Device, error) {
	sensorTypeRegex := regexp.MustCompile("(^[a-z]*)([0-9]*)")
	fullPath := hostfs.ResolveHostFS(baseDir)

	_, err := os.Stat(fullPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("hwmon path %s does not exist", fullPath)
	}

	paths, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error reading directory %s: %w", fullPath, err)
	}

	sensorList := []Device{}

	for _, path := range paths {

		name := filepath.Join(fullPath, path.Name())

		apath := name
		// This lstat call is largely to deal with the relative paths used by tests
		stat, err := os.Lstat(name)
		if err != nil {
			return nil, fmt.Errorf("error statting hwinfo path %s: %w", name, err)
		}

		if stat.Mode()&os.ModeSymlink != 0 {
			apath, err = os.Readlink(name)
			if err != nil {
				return nil, fmt.Errorf("error reading path link %s: %w", name, err)
			}
		}

		if !filepath.IsAbs(apath) && stat.Mode()&os.ModeSymlink != 0 {
			apath = filepath.Join(baseDir, apath)
		}

		sensors, err := findSensorsInPath(apath, sensorTypeRegex)
		if err != nil {
			return nil, err
		}

		namePath := filepath.Join(apath, "name")
		sensorName, err := os.ReadFile(namePath)
		if err != nil {
			return nil, fmt.Errorf("error reading sensor name file %s: %w", namePath, err)
		}
		strName := strings.TrimSpace(string(sensorName))
		sensorList = append(sensorList, Device{Name: strName, AbsPath: apath, Sensors: sensors})
	}

	if len(sensorList) == 0 {
		return nil, fmt.Errorf("no hwmon devices found in %s", fullPath)
	}

	return sensorList, nil
}

// Fetch fetches the metrics and data for the sensor.
func (s Sensor) Fetch(path string) (SensorMetrics, error) {
	// All the different sensor types have a few common fields. Fetch those first.
	// See https://www.kernel.org/doc/Documentation/hwmon/sysfs-interface
	labelName := s.getName("label")
	label, err := stringStrip(labelName, path)

	//Some sensors don't have a label, make our own
	if os.IsNotExist(err) {
		label = fmt.Sprintf("%s_%d", s.DevType.fileKey, s.SensorNum)
	} else if err != nil {
		return SensorMetrics{}, fmt.Errorf("error fetching label for %s in %s: %w", labelName, path, err)
	}

	// Not sure if we want this to be an error, since a lot of OSes, particularly stuff running inside a VM,
	// will just have this invalid hwmon entries with labels but no values. We may want this to be a log-level error instead.
	inputName := s.getName("input")
	input, err := getValueForSensor(inputName, path, s.DevType)
	if os.IsNotExist(err) {
		return SensorMetrics{}, ErrNoMetric
	} else if err != nil {
		return SensorMetrics{}, fmt.Errorf("error fetching input for %s in %s: %w", inputName, path, err)
	}

	sensorData := SensorMetrics{
		Label:      label,
		sensorType: s.DevType,
		Value:      input,
	}

	// Other special metrics for some sensors
	// We don't want to bulk fetch these with a glob or something, we'll just end up picking up a bunch of garbage
	critName := s.getName("crit")
	critVal, _ := getValueForSensor(critName, path, s.DevType)
	sensorData.Critical = critVal

	maxName := s.getName("max")
	maxVal, _ := getValueForSensor(maxName, path, s.DevType)
	sensorData.Max = maxVal

	lowestName := s.getName("lowest")
	lowestVal, _ := getValueForSensor(lowestName, path, s.DevType)
	sensorData.Lowest = lowestVal

	avgName := s.getName("average")
	avgVal, _ := getValueForSensor(avgName, path, s.DevType)
	sensorData.Average = avgVal

	return sensorData, nil
}

// Get a formatted filename
func (s Sensor) getName(file string) string {
	return fmt.Sprintf("%s%d_%s", s.DevType.fileKey, s.SensorNum, file)
}

// look for all the individual sensors in a hwmon path
func findSensorsInPath(path string, sensorRegex *regexp.Regexp) ([]Sensor, error) {
	sensorList := []Sensor{}

	//This is just to track what sensors we've found, as hwmon just dumps everything into one directory
	foundMap := map[string]bool{}

	//iterate over the files in the hwmon path
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("error reading from hwmon path %s: %w", path, err)
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		// The actual hwmon sensor files are formatted as typeNUM_filetype
		if !strings.Contains(file.Name(), "_") {
			continue
		}
		prefixes := sensorRegex.FindStringSubmatch(file.Name())
		//There should be three values here: the total match, and the two submatches for type and number
		//These directories have a lot of stuff in them, so this isn't an error.
		if len(prefixes) < 3 {
			continue
		}
		_, found := foundMap[prefixes[0]]
		if found {
			continue
		}

		st, found := getSensorType(prefixes[1])
		// Skip sensor types that we currently don't support
		if !found {
			continue
		}
		sensorNum, err := strconv.ParseInt(prefixes[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing int %s: %w", prefixes[2], err)
		}
		foundMap[prefixes[0]] = true
		sensorList = append(sensorList, Sensor{DevType: st, SensorNum: sensorNum})

	}

	return sensorList, nil
}

// Small helper function for all the boilerplate
func stringStrip(name, path string) (string, error) {
	fullpath := filepath.Join(path, name)
	raw, err := os.ReadFile(fullpath)
	// pass through file not found
	if os.IsNotExist(err) {
		return "", err
	}
	if err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// another helper that adds strconv
func stringStripInt(name, path string) (opt.Uint, error) {
	raw, err := stringStrip(name, path)
	//passthrough errors for file-not-found
	if err != nil {
		return opt.NewUintNone(), err
	}
	conv, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return opt.NewUintNone(), fmt.Errorf("error converting value %s: %w", raw, err)
	}
	return opt.UintWith(uint64(conv)), nil
}

// Another helper that's used for float64 metrics/
// This will also convert millicelsius values to celsius.
func getValueForSensor(name, path string, st SensorType) (opt.Uint, error) {
	intval, err := stringStripInt(name, path)
	if err != nil {
		return opt.NewUintNone(), fmt.Errorf("error fetching int val %s: %w", name, err)
	}

	if st == TempSensor {
		intval = millicelsiusToCelsius(intval)
	}

	return intval, nil
}

func millicelsiusToCelsius(in opt.Uint) opt.Uint {
	return opt.UintWith(in.ValueOr(0) / 1000)
}

func getSensorType(in string) (SensorType, bool) {
	var sensor SensorType
	found := true
	switch in {
	case "temp":
		sensor = TempSensor
	case "in":
		sensor = VoltSensor
	case "fan":
		sensor = FanSensor
	default:
		found = false
	}
	return sensor, found
}
