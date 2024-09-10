// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/panw"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	KBytes = 1024
	MBytes = 1024 * KBytes
	GBytes = 1024 * MBytes
)

const filesystemQuery = "<show><system><disk-space></disk-space></system></show>"

var filesystemLogger *logp.Logger

func getFilesystemEvents(m *MetricSet) ([]mb.Event, error) {
	// Set logger so all the parse functions have access
	filesystemLogger = m.logger
	var response FilesystemResponse

	output, err := m.client.Op(filesystemQuery, panw.Vsys, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("error querying filesystem info: %w", err)
	}

	if len(output) == 0 {
		return nil, fmt.Errorf("received empty output from filesystem query")
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling filesystem response: %w", err)
	}

	filesystems := getFilesystems(response.Result.Data)
	events := formatFilesystemEvents(m, filesystems)

	return events, nil
}

func getFilesystems(input string) []Filesystem {

	filesystemLogger.Debugf("getFilesystems input:\n %s", input)
	lines := strings.Split(input, "\n")
	filesystems := make([]Filesystem, 0)

	// Skip the first line which is the header:
	//
	// Example:
	// Result from the XML API call is basically a command in Linux distribution i.e., "df -h"'s output:
	//
	//	Filesystem      Size  Used Avail Use% Mounted on
	//	/dev/root       9.5G  4.0G  5.1G  44% /
	//	none            2.5G   64K  2.5G   1% /dev
	//	/dev/sda5        19G  9.1G  9.0G  51% /opt/pancfg
	//
	for _, line := range lines[1:] {
		fields := strings.Fields(line)
		if len(fields) == 6 {
			filesystem := Filesystem{
				Name:    fields[0],
				Size:    fields[1],
				Used:    fields[2],
				Avail:   fields[3],
				UsePerc: fields[4],
				Mounted: fields[5],
			}
			filesystems = append(filesystems, filesystem)
		}
	}
	return filesystems
}

func convertToBytes(field string, value string) float64 {
	if len(value) == 0 {
		filesystemLogger.Warn("convertToBytes called with empty value")
		return -1
	}

	// value, for instance for "used", can be just "0", so just return that
	if value == "0" {
		return 0
	}

	//filesystemLogger.Warnf("convertToBytes field %s, value: %s.", field, value)
	numstr := value[:len(value)-1]
	units := strings.ToLower(value[len(value)-1:])
	result, err := strconv.ParseFloat(numstr, 32)
	if err != nil {
		filesystemLogger.Warnf("parseFloat failed to parse field %s, value: %s. Error: %v", field, value, err)
		return -1
	}

	switch units {
	case "k":
		return result * KBytes
	case "m":
		return result * MBytes
	case "g":
		return result * GBytes
	default:
		// Handle values without units
		if units == "" {
			return result
		} else {
			filesystemLogger.Warnf("Unhandled units for field %s, value %s: %s", field, value, units)
			return result
		}
	}

}

func formatFilesystemEvents(m *MetricSet, filesystems []Filesystem) []mb.Event {
	if len(filesystems) == 0 {
		return nil
	}

	events := make([]mb.Event, 0, len(filesystems))
	timestamp := time.Now()
	for _, filesystem := range filesystems {
		used, err := strconv.ParseInt(filesystem.UsePerc[:len(filesystem.UsePerc)-1], 10, 64)
		if err != nil {
			filesystemLogger.Warnf("Failed to parse used percent: %v", err)
		}

		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"filesystem.name":        filesystem.Name,
				"filesystem.size":        convertToBytes("filesystem.size", filesystem.Size),
				"filesystem.used":        convertToBytes("filesystem.used", filesystem.Used),
				"filesystem.available":   convertToBytes("filesystem.available", filesystem.Avail),
				"filesystem.use_percent": used,
				"filesystem.mounted":     filesystem.Mounted,
			},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
			},
		}

		events = append(events, event)
	}

	return events
}
