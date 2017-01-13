package perf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/module/ceph"
	"os/exec"
	"strings"
)

const (
	measurement = "ceph"
	typeMon     = "monitor"
	typeOsd     = "osd"
	osdPrefix   = "ceph-osd"
	monPrefix   = "ceph-mon"
	sockSuffix  = "asok"
)

func eventsMapping(socketsList []*socket, binaryPath string) []common.MapStr {
	myEvents := []common.MapStr{}

	for _, socket := range socketsList {

		dump, err := perfDump(binaryPath, socket)
		if err != nil {
			logp.Err("An error occurred while reading sockets for getting ceph perf: %v", err)
			continue
		}

		data, err := parseDump(dump)
		if err != nil {
			logp.Err("An error occurred while parsing data for getting ceph perf: %v", err)
			continue
		}

		for tag, datapoints := range data {

			event := common.MapStr{
				formatTagName(tag): datapoints,
			}

			myEvents = append(myEvents, event)
		}
	}

	return myEvents
}

func formatTagName(oldtag string) string {

	// Replace '::' fields
	r := strings.NewReplacer("::", ".", ":.", ".", ":", ".")

	return r.Replace(oldtag)

}

func perfDump(binary string, socket *socket) (string, error) {
	var output string

	cmdArgs := []string{"--admin-daemon", socket.socket}
	if socket.sockType == typeOsd {
		cmdArgs = append(cmdArgs, "perf", "dump")
	} else if socket.sockType == typeMon {
		cmdArgs = append(cmdArgs, "perfcounters_dump")
	} else {
		return "", fmt.Errorf("[Unknown socket type] %s", socket.sockType)
	}

	if strings.Contains(binary, "docker") {
		cmdCeph := []string{"/usr/bin/ceph"}
		cmdDockerExec := append(cmdCeph, cmdArgs...)
		output = ceph.ExecInContainer(cmdDockerExec)
	} else {
		cmd := exec.Command(binary, cmdArgs...)

		var out bytes.Buffer
		cmd.Stdout = &out
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("[Error running ceph dump command] %s", stderr.String())
		}
		output = out.String()
	}

	return output, nil

}

func parseDump(dump string) (map[string]interface{}, error) {
	data := make(map[string]interface{})

	err := json.Unmarshal([]byte(dump), &data)

	return data, err

}
