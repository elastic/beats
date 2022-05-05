package hwmon

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/elastic/elastic-agent-libs/opt"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
	"github.com/elastic/go-structform"
)

var baseDir = "/sys/class/hwmon"

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

// SensorMetrics reports the actual metrics in a sensors
// This is meant to be generic for all possible sensor types,
// hence the considerable heterogeneous fields
type SensorMetrics struct {
	//Generic Fields
	Label string `struct:"label,omitempty"`

	// This field gets inserted into the map, before individual values.
	sensorType SensorType

	Critical opt.Uint `struct:"critical,omitempty"`
	Max      opt.Uint `struct:"max,omitempty"`
	Lowest   opt.Uint `struct:"lowest,omitempty"`
	Average  opt.Uint `struct:"average,omitempty"`

	// The input value of the metric. The key is overriden and set to the value of sensorType by Fold()
	Value opt.Uint `struct:"value,omitempty"`
}

// Fold implements the Folder interface for structform
// This is entirely so we can carry around a relatively simple struct that transforms into the more heavily nested event that's the standard for beats events.
func (sm *SensorMetrics) Fold(v structform.ExtVisitor) error {
	val := reflect.ValueOf(sm).Elem()
	types := reflect.TypeOf(sm).Elem()

	v.OnObjectStart(val.NumField(), structform.AnyType)

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
	err := v.OnObjectFinished()
	if err != nil {
		return fmt.Errorf("error in OnObjectFinished for %s: %w", sm.Label, err)
	}

	return nil
}

// MonData is a simple wrapper type for the map returned by ReportSensors
type MonData map[string]SensorMetrics

// ErrNoMetric specifies that no metrics were found for a given Sensor.
// This is meant to be a soft error if needed, as the slapdash nature of hwmon sysfs files
// means that we can see *_label files with no corrisponding metrics.
var ErrNoMetric = errors.New("No Metrics exist in this device")

//ReportSensors returns the metrics from all the known sensors.
//We would normally want a concrete data type here, but these metrics are so variable that we don't get much from it.
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
		//sensorName := fmt.Sprintf("%s%d", sensor.DevType, sensor.SensorNum)
		labelName := strings.ToLower(strings.Replace(data.Label, " ", "_", -1))
		metrics[labelName] = data
	}

	return metrics, nil
}

//DetectHwmon returns a list of hwmon sensors found on the system, if they exist
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
	// the hwmon device directory is just a bunch of symlinks.
	for _, path := range paths {
		name := filepath.Join(fullPath, path.Name())
		apath, err := os.Readlink(name)
		if err != nil {
			return nil, fmt.Errorf("error reading path link %s: %w", name, err)
		}

		if !filepath.IsAbs(apath) {
			//These paths are usually relative, and filepath.Join will attempt to "clean" the realtive folders once we append an absolute base path
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

	//Some directories will skip a label if there's only one sensor.
	if os.IsNotExist(err) {
		//try to fetch a name from the device symlink
		cname, err := stringStrip("name", path)
		if err != nil {
			return SensorMetrics{}, fmt.Errorf("error reading name file: %w", err)
		}
		label = cname
	} else if err != nil {
		return SensorMetrics{}, fmt.Errorf("error fetching label for %s in %s: %w", labelName, path, err)
	}

	// Not sure if we want this to be an error, since a lot of OSes, particularly stuff running inside a VM,
	// will just have this invalid hwmon entries with labels but no values. We may want this to be a log-level error instead.
	inputName := s.getName("input")
	input, err := getAndDiv(inputName, path, s.DevType)
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
	// We don't want to bulk fetch these with a glob or something, since the hwmon interface is so slapdash, we'll just end up picking up a bunch of garbage
	critName := s.getName("crit")
	critVal, _ := getAndDiv(critName, path, s.DevType)
	sensorData.Critical = critVal

	maxName := s.getName("max")
	maxVal, _ := getAndDiv(maxName, path, s.DevType)
	sensorData.Max = maxVal

	lowestName := s.getName("lowest")
	lowestVal, _ := getAndDiv(lowestName, path, s.DevType)
	sensorData.Lowest = lowestVal

	avgName := s.getName("average")
	avgVal, _ := getAndDiv(avgName, path, s.DevType)
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
func stringStripInt(name, path string) (int64, error) {
	raw, err := stringStrip(name, path)
	//passthrough errors for file-not-found
	if err != nil {
		return 0, err
	}
	conv, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("error converting value %s: %w", raw, err)
	}
	return conv, nil
}

// Another helper that's used for float64 metrics where celsius values get divided from millidegrees.
func getAndDiv(name, path string, st SensorType) (opt.Uint, error) {
	intval, err := stringStripInt(name, path)
	if err != nil {
		return opt.NewUintNone(), fmt.Errorf("error fetching int val %s: %w", name, err)
	}

	if st == TempSensor {
		intval = intval / 1000
	}

	return opt.UintWith(uint64(intval)), nil
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
