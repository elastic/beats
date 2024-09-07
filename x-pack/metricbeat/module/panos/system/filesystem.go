// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package system

import (
	"encoding/xml"
	"strings"
	"time"

	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

func getFilesystemEvents(m *MetricSet) ([]mb.Event, error) {
	query := "<show><system><disk-space></disk-space></system></show>"

	var response FilesystemResponse

	output, err := m.client.Op(query, vsys, nil, nil)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	err = xml.Unmarshal(output, &response)
	if err != nil {
		m.logger.Error("Error: %s", err)
		return nil, err
	}

	filesystems := getFilesystems(response.Result.Data)
	events := formatFilesytemEvents(m, filesystems)

	return events, nil
}

/*
Result from the XML API call is basically linux df -h output:
Filesystem      Size  Used Avail Use% Mounted on
/dev/root       9.5G  4.0G  5.1G  44% /
none            2.5G   64K  2.5G   1% /dev
/dev/sda5        19G  9.1G  9.0G  51% /opt/pancfg
/dev/sda6       7.6G  3.1G  4.2G  43% /opt/panrepo
tmpfs           2.5G  399M  2.1G  16% /dev/shm
cgroup_root     2.5G     0  2.5G   0% /cgroup
/dev/sda8       173G   63G  102G  39% /opt/panlogs
tmpfs            12M   44K   12M   1% /opt/pancfg/mgmt/ssl/private
*/
func getFilesystems(input string) []Filesystem {
	lines := strings.Split(input, "\n")
	filesystems := make([]Filesystem, 0)

	// Skip the first line which is the header
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

func formatFilesytemEvents(m *MetricSet, filesystems []Filesystem) []mb.Event {
	events := make([]mb.Event, 0, len(filesystems))

	currentTime := time.Now()

	for _, filesystem := range filesystems {
		event := mb.Event{MetricSetFields: mapstr.M{
			"name":        filesystem.Name,
			"size":        filesystem.Size,
			"used":        filesystem.Used,
			"available":   filesystem.Avail,
			"use_percent": filesystem.UsePerc,
			"mounted":     filesystem.Mounted,
		},
			RootFields: mapstr.M{
				"observer.ip":     m.config.HostIp,
				"host.ip":         m.config.HostIp,
				"observer.vendor": "Palo Alto",
				"observer.type":   "firewall",
				"@Timestamp":      currentTime,
			}}

		events = append(events, event)
	}

	return events

}
