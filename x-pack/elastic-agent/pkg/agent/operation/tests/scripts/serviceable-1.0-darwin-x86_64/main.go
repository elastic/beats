// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	protobuf "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"gopkg.in/yaml.v2"

	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

func main() {
	srvPort, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	f, _ := os.OpenFile(filepath.Join(os.TempDir(), "testing.out"), os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	f.WriteString("starting \n")
	ctx, cancel := context.WithCancel(context.Background())
	s := &configServer{
		f:      f,
		ctx:    ctx,
		cancel: cancel,
	}
	f.WriteString(fmt.Sprintf("reading creds from port: %d\n", srvPort))
	client, err := clientFromNet(srvPort, s)
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

	s.client.Status(proto.StateObserved_HEALTHY, "Running", map[string]interface{}{
		"status":  proto.StateObserved_HEALTHY,
		"message": "Running",
	})
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
	TestFile string `config:"TestFile" yaml:"TestFile"`
}

func getCreds(port int) (*proto.ConnInfo, error) {
	c, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return nil, err
	}
	defer c.Close()
	buf := make([]byte, 1024*1024)
	n, err := c.Read(buf)
	if err != nil {
		return nil, err
	}
	var connInfo proto.ConnInfo
	err = protobuf.Unmarshal(buf[:n], &connInfo)
	if err != nil {
		return nil, err
	}
	return &connInfo, nil
}

func clientFromNet(port int, impl client.StateInterface, actions ...client.Action) (client.Client, error) {
	connInfo, err := getCreds(port)
	if err != nil {
		return nil, err
	}
	cert, err := tls.X509KeyPair(connInfo.PeerCert, connInfo.PeerKey)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(connInfo.CaCert)
	trans := credentials.NewTLS(&tls.Config{
		ServerName:   connInfo.ServerName,
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
	})
	return client.New(connInfo.Addr, connInfo.Token, impl, actions, grpc.WithTransportCredentials(trans)), nil
}
