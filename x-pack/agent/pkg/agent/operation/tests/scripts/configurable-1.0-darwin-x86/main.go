// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/plugin/server"
	"github.com/elastic/beats/v7/x-pack/agent/pkg/core/remoteconfig/grpc"
)

func main() {
	f, _ := os.OpenFile(filepath.Join(os.TempDir(), "testing.out"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	f.WriteString("starting \n")
	s := &configServer{}
	if err := server.NewGrpcServer(os.Stdin, s); err != nil {
		f.WriteString(err.Error())
		panic(err)
	}
	f.WriteString("finished \n")
}

type configServer struct {
}

// TestConfig is a configuration for testing Config calls
type TestConfig struct {
	TestFile string `config:"TestFile" yaml:"TestFile"`
}

func (*configServer) Config(ctx context.Context, req *grpc.ConfigRequest) (*grpc.ConfigResponse, error) {
	cfgString := req.GetConfig()

	testCfg := &TestConfig{}
	if err := yaml.Unmarshal([]byte(cfgString), &testCfg); err != nil {
		return &grpc.ConfigResponse{}, err
	}

	if testCfg.TestFile != "" {
		tf, err := os.Create(testCfg.TestFile)
		if err != nil {
			return &grpc.ConfigResponse{}, err
		}

		err = tf.Close()
		if err != nil {
			return &grpc.ConfigResponse{}, err
		}
	}

	return &grpc.ConfigResponse{}, nil
}

// Status return ok.
func (*configServer) Status(ctx context.Context, req *grpc.StatusRequest) (*grpc.StatusResponse, error) {
	return &grpc.StatusResponse{Status: "ok"}, nil
}
