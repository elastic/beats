// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fbtest

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	fbroot "github.com/elastic/beats/v7/x-pack/filebeat/cmd"

	// initialize the plugin system before libbeat does, so we can overwrite it properly
	_ "github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/beats/v7/x-pack/libbeat/management/tests"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

var expectedFBStreams = &proto.UnitExpectedConfig{
	DataStream: &proto.DataStream{
		Namespace: "default",
	},
	Type:     "log",
	Id:       "logfile-system-default-system",
	Name:     "system-1",
	Revision: 1,
	Meta: &proto.Meta{
		Package: &proto.Package{
			Name:    "system",
			Version: "1.17.0",
		},
	},
}

func TestFilebeat(t *testing.T) {
	filebeatCmd := fbroot.Filebeat()
	tests.InitBeatsForTest(t, filebeatCmd)
	var fbStreams = []*proto.Stream{
		{
			Id: "logfile-system.syslog-default-system",
			DataStream: &proto.DataStream{
				Dataset: "system.syslog",
				Type:    "logs",
			},
			Source: tests.RequireNewStruct(map[string]interface{}{
				"paths":         []interface{}{"./testdata/messages"},
				"exclude_files": []interface{}{".gz$"},
				"multiline": map[string]interface{}{
					"pattern": `^\s`,
					"match":   "after",
				},
			}),
		},
		{
			Id: "logfile-system.auth-default-system",
			DataStream: &proto.DataStream{
				Dataset: "system.auth",
				Type:    "logs",
			},
			Source: tests.RequireNewStruct(map[string]interface{}{
				"paths":         []interface{}{"./testdata/secure*"},
				"exclude_files": []interface{}{".gz$"},
				"multiline": map[string]interface{}{
					"pattern": `^\s`,
					"match":   "after",
				},
			}),
		},
	}

	expectedFBStreams.Streams = fbStreams

	outPath, server := tests.SetupTestEnv(t, expectedFBStreams, time.Second*30)

	defer func() {
		err := os.RemoveAll(outPath)
		require.NoError(t, err)
	}()

	go func() {
		t.Logf("Running beats...")
		err := filebeatCmd.Execute()
		require.NoError(t, err)
	}()

	// Wait until we've read from all log sources
	t.Logf("waiting for events...")
	for {
		time.Sleep(time.Second)
		//t.Logf("checking for stop condition in %s", outputFilesPath)
		events := tests.ReadLogLines(t, outPath)
		if events >= 90 {
			t.Logf("stopping filebeat after %d events", events)
			server.Client.Stop()
			server.Srv.Stop()
			break
		}
	}

	t.Logf("Reading events...")
	events := tests.ReadEvents(t, outPath)
	// Look for processors
	expectedMetaValuesSyslog := map[string]interface{}{
		// Processors created by
		"@metadata.input_id":    "logfile-system-default-system",
		"@metadata.stream_id":   "logfile-system.syslog-default-system",
		"agent.id":              "test-agent",
		"data_stream.dataset":   "system.syslog",
		"data_stream.namespace": "default",
		"data_stream.type":      "logs",
	}
	tests.ValuesExist(t, expectedMetaValuesSyslog, events, tests.ONCE, "expectedMetaValuesSyslog")

	expectedMetaValuesAuth := map[string]interface{}{
		// Processors created by
		"@metadata.input_id":  "logfile-system-default-system",
		"@metadata.stream_id": "logfile-system.auth-default-system",
		"agent.id":            "test-agent",
		"data_stream.dataset": "system.auth",
	}
	tests.ValuesExist(t, expectedMetaValuesAuth, events, tests.ONCE, "expectedMetaValuesAuth")

	expectedLogValues := map[string]interface{}{
		"log.file.path": nil,
		"message":       nil,
	}
	tests.ValuesExist(t, expectedLogValues, events, tests.ONCE, "expectedLogValues")
}
