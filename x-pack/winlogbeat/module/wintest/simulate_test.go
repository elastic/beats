// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

// Windows is excluded not because the tests won't pass on Windows in general,
// but because they won't pass on Windows in a VM — where we are using this — due
// to the VM inception problem.
//
//go:build !windows

package wintest_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	devtools "github.com/elastic/beats/v7/dev-tools/mage"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/winlogbeat/module"
	"github.com/elastic/beats/v7/x-pack/winlogbeat/module/wintest"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

// ecsVersion is the expected ECS version for testing purposes.
// Change this when ECS version is bumped.
const ecsVersion = "1.12.0"

func TestSimulate(t *testing.T) {
	const (
		host        = "http://localhost:9200"
		user        = "admin"
		pass        = "testing"
		indexPrefix = "winlogbeat-test"
		pipeline    = "powershell"
		pattern     = "testdata/*.evtx.json"
	)

	done, _, err := wintest.Docker(".", "test", testing.Verbose())
	if err != nil {
		t.Fatal(err)
	}
	if *wintest.KeepRunning {
		fmt.Fprintln(os.Stdout, "docker-compose", "-p", devtools.DockerComposeProjectName(), "rm", "--stop", "--force")
	}
	t.Cleanup(func() {
		stop := !*wintest.KeepRunning
		err = done(stop)
		if err != nil {
			t.Errorf("unexpected error during cleanup: %v", err)
		}
	})

	// Currently we are using mixed API because beats is using the old ES API package,
	// while SimulatePipeline is using the official v8 client package.
	conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
		URL:              host,
		Username:         user,
		Password:         pass,
		CompressionLevel: 3,
		Transport:        httpcommon.HTTPTransportSettings{Timeout: time.Minute},
	})
	if err != nil {
		t.Fatalf("unexpected error making connection: %v", err)
	}
	defer conn.Close()

	err = conn.Connect()
	if err != nil {
		t.Fatalf("unexpected error making connection: %v", err)
	}

	info := beat.Info{
		IndexPrefix: indexPrefix,
		Version:     version.GetDefaultVersion(),
	}
	loaded, err := module.UploadPipelines(info, conn, true)
	if err != nil {
		t.Errorf("unexpected error uploading pipelines: %v", err)
	}
	wantPipelines := []string{
		"powershell",
		"powershell_operational",
		"routing",
		"security",
		"sysmon",
	}
	if len(loaded) != len(wantPipelines) {
		t.Fatalf("unexpected number of loaded pipelines: got:%d want:%d", len(loaded), len(wantPipelines))
	}
	want := regexp.MustCompile(`^` + indexPrefix + `-.*-(` + strings.Join(wantPipelines, "|") + `)$`)
	pipelines := make(map[string]string)
	for _, p := range loaded {
		m := want.FindAllStringSubmatch(p, -1)
		pipelines[m[0][1]] = p
	}
	_, ok := pipelines[pipeline]
	if !ok {
		t.Fatalf("failed to upload %q", pipeline)
	}

	paths, err := filepath.Glob(pattern)
	if err != nil {
		t.Fatalf("failed to expand glob pattern %q", pattern)
	}
	cases, err := wintest.SimulatePipeline(host, user, pass, pipelines[pipeline], paths)
	if err != nil {
		t.Fatalf("unexpected error running simulate: %v", err)
	}
	for _, k := range cases {
		name := filepath.Base(k.Path)
		t.Run(name, func(t *testing.T) {
			if k.Err != nil {
				t.Errorf("unexpected error: %v", k.Err)
				return
			}
			for i := range k.Collected {
				t.Logf("%s %d:\ncollected:\n%s\n\nprocessed:\n%s\n\n", name, i, k.Collected[i], k.Processed[i])

				// Check that the ECS version is in place in the processed event.
				// This is not present in the original evtx.json files and so is
				// a robust indicator that the event has passed through the
				// processor pipeline.
				var event struct {
					ECS struct {
						Version string
					}
				}
				err := json.Unmarshal(k.Processed[i], &event)
				if err != nil {
					t.Errorf("unexpected error unmarshaling ECS version: %v", err)
					continue
				}
				if event.ECS.Version != ecsVersion {
					t.Errorf("unexpected ECS version: want:%q got:%q", ecsVersion, event.ECS.Version)
				}

				// Check for errors. There are none in this set of events and we cannot
				// guarantee that later changes in the pipelines will not remove errors;
				// that being the point of this game.
				err = wintest.ErrorMessage(k.Processed[i])
				if err != nil {
					t.Errorf("unexpected ingest error for event %d: %v", i, err)
				}
			}
		})
	}
}
