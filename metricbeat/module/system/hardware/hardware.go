package hardware

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"gopkg.in/yaml.v2"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "hardware", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	hardwareQuery        []queryKey
	formatQuery          configYaml
	hardware             common.MapStr
	hardwareMonitorQuery []win32MonitorID
}

type queryKey struct {
	Type         string
	Name         string
	DeviceID     string
	Description  string
	Manufacturer string
	Output       innerConfigFormat
	Index        int
}

type win32MonitorID struct {
	UserFriendlyName  []int8
	YearOfManufacture int
	Output            innerConfigFormat
	Type              string
	Index             int
}

var newQuery = []queryKey{}
var monitorQuery = []win32MonitorID{}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system hardware metricset is beta.")
	var cfg configYaml
	readFile(&cfg)

	for _, value := range cfg.Query {
		// Special ability to handle WmiMonitorID
		if value.TypeOf != "WmiMonitorID" {
			var dst []queryKey
			wmi.Query("Select * from "+value.TypeOf, &dst)
			for i, v := range dst {
				newQuery = append(newQuery, queryKey{Name: v.Name, Description: v.Description, DeviceID: v.DeviceID, Manufacturer: v.Manufacturer, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}
		} else {
			var dst []win32MonitorID
			err := wmi.QueryNamespace("select * from "+value.TypeOf, &dst, "root\\WMI")
			if err != nil {
				log.Println(err)
			}
			for i, v := range dst {
				monitorQuery = append(monitorQuery, win32MonitorID{UserFriendlyName: v.UserFriendlyName, YearOfManufacture: v.YearOfManufacture, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}

		}

	}

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet:        base,
		hardwareQuery:        newQuery,
		hardwareMonitorQuery: monitorQuery,
		hardware:             common.MapStr{},
	}, nil
}

func getData() {
	resp, err := http.Get("https://jsonplaceholder.typicode.com/users")
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println(string(body))
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	metricSetFields := common.MapStr{}
	for _, hard := range m.hardwareQuery {
		rootFields := common.MapStr{
			"Type":         hard.Type,
			"Name":         hard.Name,
			"Description":  hard.Description,
			"Manufacturer": hard.Manufacturer,
			"DeviceID":     hard.DeviceID,
		}
		if hard.Output.UseConst == true {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"data": rootFields,
				},
			})
		}

		if hard.Output.UseType == true {
			if len(m.hardwareQuery) == 1 {
				metricSetFields[hard.Type] = rootFields
			} else if hard.Index == 1 {
				metricSetFields[hard.Type] = common.MapStr{
					strconv.Itoa(hard.Index): rootFields,
				}
			} else {
				newMap := metricSetFields[hard.Type].(common.MapStr)
				newMap[strconv.Itoa(hard.Index)] = rootFields
			}
		}
	}
	for _, hard := range m.hardwareMonitorQuery {
		rootFields := common.MapStr{
			"Type":             hard.Type,
			"Name":             b2s(hard.UserFriendlyName),
			"ManufacturerYear": hard.YearOfManufacture,
		}
		if hard.Output.UseConst == true {
			report.Event(mb.Event{
				MetricSetFields: common.MapStr{
					"data": rootFields,
				},
			})
		}
		if hard.Output.UseType == true {
			if len(m.hardwareMonitorQuery) == 1 {
				metricSetFields[hard.Type] = rootFields
			} else if hard.Index == 1 {
				metricSetFields[hard.Type] = common.MapStr{
					strconv.Itoa(hard.Index): rootFields,
				}
			} else {
				newMap := metricSetFields[hard.Type].(common.MapStr)
				newMap[strconv.Itoa(hard.Index)] = rootFields
			}
		}

	}

	if len(metricSetFields) > 0 {
		var event mb.Event
		event.MetricSetFields = metricSetFields
		report.Event(event)
	}

	return nil
}

type configYaml struct {
	Query  []innerConfig     `yaml:"hardware_query"`
	Format innerConfigFormat `yaml:"output_format"`
}

type innerConfig struct {
	TypeOf string `yaml:"type"`
	Name   string `yaml:"name"`
}

type innerConfigFormat struct {
	UseType  bool `yaml:"use_type_as_key"`
	UseConst bool `yaml:"use_constant_key"`
}

func readFile(cfg *configYaml) {
	f, err := os.Open("hardware.yml")
	if err != nil {
		processError(err)
	}
	defer f.Close()

	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(cfg)
	if err != nil {
		processError(err)
	}
}

func processError(err error) {
	fmt.Println(err)
	os.Exit(2)
}

func b2s(bs []int8) string {
	b := make([]byte, len(bs))
	for i, v := range bs {
		b[i] = byte(v)
	}
	return string(b)
}
