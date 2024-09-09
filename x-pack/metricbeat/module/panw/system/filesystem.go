// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const filesystemQuery = "<show><system><disk-space></disk-space></system></show>"

func getFilesystemEvents(m *MetricSet) ([]mb.Event, error) {

	var response FilesystemResponse

	output, err := m.client.Op(filesystemQuery, vsys, nil, nil)
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

func formatFilesystemEvents(m *MetricSet, filesystems []Filesystem) []mb.Event {
	if len(filesystems) == 0 {
		return nil
	}

	events := make([]mb.Event, 0, len(filesystems))
	timestamp := time.Now()

	for _, filesystem := range filesystems {
		event := mb.Event{
			Timestamp: timestamp,
			MetricSetFields: mapstr.M{
				"filesystem.name":        filesystem.Name,
				"filesystem.size":        filesystem.Size,
				"filesystem.used":        filesystem.Used,
				"filesystem.available":   filesystem.Avail,
				"filesystem.use_percent": filesystem.UsePerc,
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
