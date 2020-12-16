// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func main() {
	f, _ := os.OpenFile(filepath.Join(os.TempDir(), "testing.out"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	f.WriteString("starting \n")
	ctx, cancel := context.WithCancel(context.Background())
	s := &configServer{
		f:      f,
		ctx:    ctx,
		cancel: cancel,
	}
	client, err := client.NewFromReader(os.Stdin, s)
	if err != nil {
		f.WriteString(err.Error())
		panic(err)
	}
	s.client = client
	err = client.Start(ctx)
	if err != nil {
		f.WriteString(err.Error())
		panic(err)
	}
	<-ctx.Done()
	f.WriteString("finished \n")
}

type configServer struct {
	f      *os.File
	ctx    context.Context
	cancel context.CancelFunc
	client client.Client
}

func (s *configServer) OnConfig(cfgString string) {
	s.client.Status(proto.StateObserved_CONFIGURING, "Writing config file", nil)

	testCfg := &TestConfig{}
	if err := yaml.Unmarshal([]byte(cfgString), &testCfg); err != nil {
		s.client.Status(proto.StateObserved_FAILED, fmt.Sprintf("Failed to unmarshall config: %s", err), nil)
		return
	}

	if testCfg.TestFile != "" {
		tf, err := os.Create(testCfg.TestFile)
		if err != nil {
			s.client.Status(proto.StateObserved_FAILED, fmt.Sprintf("Failed to create file %s: %s", testCfg.TestFile, err), nil)
			return
		}

		err = tf.Close()
		if err != nil {
			s.client.Status(proto.StateObserved_FAILED, fmt.Sprintf("Failed to close file %s: %s", testCfg.TestFile, err), nil)
			return
		}
	}

	if testCfg.Crash {
		os.Exit(2)
	}

	if testCfg.Status != nil {
		s.client.Status(*testCfg.Status, "Custom status", map[string]interface{}{
			"status":  *testCfg.Status,
			"message": "Custom status",
		})
	} else {
		s.client.Status(proto.StateObserved_HEALTHY, "Running", map[string]interface{}{
			"status":  proto.StateObserved_HEALTHY,
			"message": "Running",
		})
	}
}

func (s *configServer) OnStop() {
	s.client.Status(proto.StateObserved_STOPPING, "Stopping", nil)
	s.cancel()
}

func (s *configServer) OnError(err error) {
	s.f.WriteString(err.Error())
}

// TestConfig is a configuration for testing Config calls
type TestConfig struct {
	TestFile string                      `config:"TestFile" yaml:"TestFile"`
	Status   *proto.StateObserved_Status `config:"Status" yaml:"Status"`
	Crash    bool                        `config:"Crash" yaml:"Crash"`
}
