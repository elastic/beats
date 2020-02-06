package hardware

import (
	"log"
	"strconv"

	"github.com/StackExchange/wmi"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/system/hardware/util"
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
	formatQuery          util.ConfigYaml
	hardware             common.MapStr
	hardwareMonitorQuery []queryKey
}

// Define the fields that return from the query
type queryKey struct {
	Type              string
	Name              string
	DeviceID          string
	Description       string
	Manufacturer      string
	UserFriendlyName  []int8
	YearOfManufacture int
	Output            util.InnerConfigFormat
	Index             int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system hardware metricset is beta.")
	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}
	var newQuery = []queryKey{}
	var monitorQuery = []queryKey{}

	return &MetricSet{
		BaseMetricSet:        base,
		hardwareQuery:        newQuery,
		hardwareMonitorQuery: monitorQuery,
		hardware:             common.MapStr{},
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	// Read the data from the hardware.yml file
	var cfg util.ConfigYaml
	util.ReadFile(&cfg)
	for _, value := range cfg.Query {
		// Regular wmi query with default namespace
		if value.TypeOf != "WmiMonitorID" {
			var dst []queryKey
			wmi.Query("Select * from "+value.TypeOf, &dst)
			for i, v := range dst {
				m.hardwareQuery = append(m.hardwareQuery, queryKey{Name: v.Name, Description: v.Description, DeviceID: v.DeviceID, Manufacturer: v.Manufacturer, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}
		} else {
			// Special ability to handle WmiMonitorID on root\\WMI namespace
			var dst []queryKey
			err := wmi.QueryNamespace("select * from "+value.TypeOf, &dst, "root\\WMI")
			if err != nil {
				log.Println(err)
			}
			for i, v := range dst {
				m.hardwareMonitorQuery = append(m.hardwareMonitorQuery, queryKey{UserFriendlyName: v.UserFriendlyName, YearOfManufacture: v.YearOfManufacture, Type: value.Name, Output: cfg.Format, Index: i + 1})
			}
		}
	}
	metricSetFields := common.MapStr{}
	for _, hard := range m.hardwareQuery {
		// Define the regular output
		rootFields := common.MapStr{
			"type":         hard.Type,
			"name":         hard.Name,
			"description":  hard.Description,
			"manufacturer": hard.Manufacturer,
			"deviceID":     hard.DeviceID,
			"index":        hard.Index,
		}
		sendEventHardware(hard, m.hardwareQuery, rootFields, metricSetFields, report)
	}
	for _, hard := range m.hardwareMonitorQuery {
		// Define the MonitorId output
		rootFields := common.MapStr{
			"type":             hard.Type,
			"name":             util.B2s(hard.UserFriendlyName),
			"manufacturerYear": hard.YearOfManufacture,
			"index":            hard.Index,
		}
		sendEventHardware(hard, m.hardwareMonitorQuery, rootFields, metricSetFields, report)
	}

	if len(metricSetFields) > 0 {
		var event mb.Event
		event.MetricSetFields = metricSetFields
		report.Event(event)
	}

	return nil
}

func sendEventHardware(hard queryKey, hardware []queryKey, rootFields common.MapStr, metricSetFields common.MapStr, report mb.ReporterV2) {
	// Set the key to data
	if hard.Output.UseConst == true {
		report.Event(mb.Event{
			MetricSetFields: common.MapStr{
				"data": rootFields,
			},
		})
	}
	if hard.Output.UseType == true {
		// If there is only one item
		if hard.Index == 1 {
			metricSetFields[hard.Type] = common.MapStr{
				strconv.Itoa(hard.Index): rootFields,
			}
		} else {
			// If there is more then one item
			newMap := metricSetFields[hard.Type].(common.MapStr)
			newMap[strconv.Itoa(hard.Index)] = rootFields
		}
	}
}
