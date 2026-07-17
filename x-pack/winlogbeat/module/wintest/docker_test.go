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
	"bytes"
	"context"
	"io"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/esleg/eslegclient"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/beats/v7/winlogbeat/module"
	"github.com/elastic/beats/v7/x-pack/winlogbeat/module/wintest"
	"github.com/elastic/elastic-agent-libs/logp/logptest"
	"github.com/elastic/elastic-agent-libs/transport/httpcommon"

	// Enable pipelines.
	_ "github.com/elastic/beats/v7/x-pack/winlogbeat/module"
)

func TestDocker(t *testing.T) {
	const (
		host        = "http://localhost:9200"
		user        = "admin"
		pass        = "testing"
		indexPrefix = "winlogbeat-test"
	)

	done, _, err := wintest.Docker(".", "test", testing.Verbose())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		const stop = false
		err = done(stop)
		if err != nil {
			t.Errorf("unexpected error during cleanup: %v", err)
		}
	})

	resp, err := getStatus(host, user, pass)
	if err != nil {
		t.Errorf("unexpected error querying elasticsearch:%v", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		t.Errorf("unexpected error copying buffer: %v", err)
	}

	got := strings.TrimSpace(buf.String())
	// Single-node test clusters report yellow when indices have unassigned
	// replica shards. Both green and yellow indicate a functional cluster.
	if got != "green" && got != "yellow" {
		t.Fatalf("unexpected cluster health status: got:%q want:green or yellow", got)
	}

	t.Run("UploadPipelines", func(t *testing.T) {
		conn, err := eslegclient.NewConnection(eslegclient.ConnectionSettings{
			URL:              host,
			Username:         user,
			Password:         pass,
			CompressionLevel: 3,
			Transport:        httpcommon.HTTPTransportSettings{Timeout: time.Minute},
		}, logptest.NewTestingLogger(t, ""))
		if err != nil {
			t.Fatalf("unexpected error making connection: %v", err)
		}
		defer conn.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		err = conn.Connect(ctx)
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
			"security_standard",
			"sysmon",
		}
		if len(loaded) != len(wantPipelines) {
			t.Errorf("unexpected number of loaded pipelines: got:%d want:%d", len(loaded), len(wantPipelines))
		}
		want := regexp.MustCompile(`^` + indexPrefix + `-.*-(?:` + strings.Join(wantPipelines, "|") + `)$`)
		for _, p := range loaded {
			if !want.MatchString(p) {
				t.Errorf("unexpected pipeline ID: %v", p)
			}
		}
	})
}

func getStatus(host, user, pass string) (*http.Response, error) {
	// To match the condition in the docker-compose file:
	//  curl -u admin:testing -s http://localhost:9200/_cat/health?h=status | grep -q green
	req, err := http.NewRequestWithContext(context.Background(), "GET", host+"/_cat/health?h=status", nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(user, pass)
	return http.DefaultClient.Do(req)
}
