// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build integration

package integration

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	protobuf "google.golang.org/protobuf/proto"

	"github.com/elastic/beats/v7/libbeat/tests/integration"
	"github.com/elastic/beats/v7/x-pack/libbeat/management"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/client/mock"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// TestInputReloadUnderElasticAgent will start a Filebeat and cause the input
// reload issue described on https://github.com/elastic/beats/issues/33653.
// In short, a new input for a file needs to be started while there are still
// events from that file in the publishing pipeline, effectively keeping
// the harvester status as `finished: false`, which prevents the new input
// from starting.
//
// This tests ensures Filebeat can gracefully recover from this situation
// and will eventually re-start harvesting the file.
//
// In case of a test failure the directory with Filebeat logs and
// all other supporting files will be kept on build/integration-tests.
//
// Run the tests with -v flag to print the temporary folder used.
func TestInputReloadUnderElasticAgent(t *testing.T) {
	// First things first, ensure ES is running and we can connect to it.
	// If ES is not running, the test will timeout and the only way to know
	// what caused it is going through Filebeat's logs.
	integration.EnsureESIsRunning(t)

	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	logFilePath := filepath.Join(filebeat.TempDir(), "flog.log")
	generateLogFile(t, logFilePath)
	var units = [][]*proto.UnitExpected{
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "default",
					Type: "elasticsearch",
					Name: "elasticsearch",
					Source: integration.RequireNewStruct(t,
						map[string]interface{}{
							"type":                 "elasticsearch",
							"hosts":                []interface{}{"http://localhost:9200"},
							"username":             "admin",
							"password":             "testing",
							"protocol":             "http",
							"enabled":              true,
							"allow_older_versions": true,
						}),
				},
			},
			{
				Id:             "input-unit-1",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "log-input",
					Type: "log",
					Name: "log",
					Streams: []*proto.Stream{
						{
							Id: "log-input-1",
							Source: integration.RequireNewStruct(t, map[string]interface{}{
								"enabled": true,
								"type":    "log",
								"paths":   []interface{}{logFilePath},
							}),
						},
					},
				},
			},
		},
		{
			{
				Id:             "output-unit",
				Type:           proto.UnitType_OUTPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "default",
					Type: "elasticsearch",
					Name: "elasticsearch",
					Source: integration.RequireNewStruct(t,
						map[string]interface{}{
							"type":                 "elasticsearch",
							"hosts":                []interface{}{"http://localhost:9200"},
							"username":             "admin",
							"password":             "testing",
							"protocol":             "http",
							"enabled":              true,
							"allow_older_versions": true,
						}),
				},
			},
			{
				Id:             "input-unit-2",
				Type:           proto.UnitType_INPUT,
				ConfigStateIdx: 1,
				State:          proto.State_HEALTHY,
				LogLevel:       proto.UnitLogLevel_DEBUG,
				Config: &proto.UnitExpectedConfig{
					Id:   "log-input",
					Type: "log",
					Name: "log",
					Streams: []*proto.Stream{
						{
							Id: "log-input-2",
							Source: integration.RequireNewStruct(t, map[string]interface{}{
								"enabled": true,
								"type":    "log",
								"paths":   []interface{}{logFilePath},
							}),
						},
					},
				},
			},
		},
	}

	// Once the desired state is reached (aka Filebeat finished applying
	// the policy changes) we still wait for a little bit before sending
	// another policy. This will allow the input to run and get some data
	// into the publishing pipeline.
	//
	// nextState is a helper function that will keep cycling through both
	// elements of the `units` slice. Once one is fully applied, we wait
	// at least 10s then send the next one.
	idx := 0
	waiting := false
	when := time.Now()
	nextState := func() {
		if waiting {
			if time.Now().After(when) {
				idx = (idx + 1) % len(units)
				waiting = false
				return
			}
			return
		}
		waiting = true
		when = time.Now().Add(10 * time.Second)
	}
	server := &mock.StubServerV2{
		// The Beat will call the check-in function multiple times:
		// - At least once at startup
		// - At every state change (starting, configuring, healthy, etc)
		// for every Unit.
		//
		// Because of that we can't rely on the number of times it is called
		// we need some sort of state machine to handle when to send the next
		// policy and when to just re-send the current one.
		//
		// If the Elastic-Agent wants the Beat to keep running the same policy,
		// it will just keep re-sending it every time the Beat calls the check-in
		// method.
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if management.DoesStateMatch(observed, units[idx], 0) {
				nextState()
			}
			for _, unit := range observed.GetUnits() {
				if state := unit.GetState(); !(state == proto.State_HEALTHY || state != proto.State_CONFIGURING || state == proto.State_STARTING) {
					t.Fatalf("Unit '%s' is not healthy, state: %s", unit.GetId(), unit.GetState().String())
				}
			}
			return &proto.CheckinExpected{
				Units: units[idx],
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	require.NoError(t, server.Start())
	t.Cleanup(server.Stop)

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	// waitDeadlineOr5Mins looks at the test deadline
	// and returns a reasonable value of waiting for a
	// condition to be met. The possible values are:
	// - if no test deadline is set, return 5 minutes
	// - if a deadline is set and there is less than
	//   0.5 second left, return the time left
	// - otherwise return the time left minus 0.5 second.
	waitDeadlineOr5Min := func() time.Duration {
		deadline, deadileSet := t.Deadline()
		if deadileSet {
			left := deadline.Sub(time.Now())
			final := left - 500*time.Millisecond
			if final <= 0 {
				return left
			}
			return final
		}
		return 5 * time.Minute
	}

	require.Eventually(t, func() bool {
		return filebeat.LogContains("Can only start an input when all related states are finished")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'Can only start an input when all related states are finished' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("file 'flog.log' is not finished, will retry starting the input soon")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'file 'flog.log' is not finished, will retry starting the input soon' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("ForceReload set to TRUE")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'ForceReload set to TRUE' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("Reloading Beats inputs because forceReload is true")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'Reloading Beats inputs because forceReload is true' not found on Filebeat logs")

	require.Eventually(t, func() bool {
		return filebeat.LogContains("ForceReload set to FALSE")
	}, waitDeadlineOr5Min(), 100*time.Millisecond,
		"String 'ForceReload set to FALSE' not found on Filebeat logs")
}

// TestFailedOutputReportsUnhealthy ensures that if an output
// fails to start and returns an error, the manager will set it
// as failed and the inputs will not be started, which means
// staying on the started state.
func TestFailedOutputReportsUnhealthy(t *testing.T) {
	// First things first, ensure ES is running and we can connect to it.
	// If ES is not running, the test will timeout and the only way to know
	// what caused it is going through Filebeat's logs.
	integration.EnsureESIsRunning(t)
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	finalStateReached := atomic.Bool{}
	var units = []*proto.UnitExpected{
		{
			Id:             "output-unit-borken",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_FAILED,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "logstash",
				Name: "logstash",
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"type":    "logstash",
						"invalid": "configuration",
					}),
			},
		},
		// Also add an input unit to make sure it never leaves the
		// starting state
		{
			Id:             "input-unit",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 1,
			State:          proto.State_STARTING,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "log-input",
				Type: "log",
				Name: "log",
				Streams: []*proto.Stream{
					{
						Id: "log-input",
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"enabled": true,
							"type":    "log",
							"paths":   "/tmp/foo",
						}),
					},
				},
			},
		},
	}

	server := &mock.StubServerV2{
		// The Beat will call the check-in function multiple times:
		// - At least once at startup
		// - At every state change (starting, configuring, healthy, etc)
		// for every Unit.
		//
		// So we wait until the state matches the desired state
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			if management.DoesStateMatch(observed, units, 0) {
				finalStateReached.Store(true)
			}

			return &proto.CheckinExpected{
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	require.NoError(t, server.Start())

	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
	)

	require.Eventually(t, func() bool {
		return finalStateReached.Load()
	}, 30*time.Second, 100*time.Millisecond, "Output unit did not report unhealthy")

	t.Cleanup(server.Stop)
}

func TestRecoverFromInvalidOutputConfiguration(t *testing.T) {
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat.test",
	)

	// Having the log file enables the inputs to start, while it is not
	// strictly necessary for testing output issues, it allows for the
	// input to start which creates a more realistic test case and
	// can help uncover other issues in the startup/shutdown process.
	logFilePath := filepath.Join(filebeat.TempDir(), "flog.log")
	generateLogFile(t, logFilePath)

	logLevel := proto.UnitLogLevel_INFO
	filestreamInputHealthy := proto.UnitExpected{
		Id:             "input-unit-healthy",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "filestream-input",
			Type: "filestream",
			Name: "filestream-input-healty",
			Streams: []*proto.Stream{
				{
					Id: "filestream-input-id",
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":      "filestream-stream-input-id",
						"enabled": true,
						"type":    "filestream",
						"paths":   logFilePath,
					}),
				},
			},
		},
	}

	filestreamInputStarting := proto.UnitExpected{
		Id:             "input-unit-2",
		Type:           proto.UnitType_INPUT,
		ConfigStateIdx: 1,
		State:          proto.State_STARTING,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "filestream-input",
			Type: "filestream",
			Name: "filestream-input-starting",
			Streams: []*proto.Stream{
				{
					Id: "filestream-input-id",
					Source: integration.RequireNewStruct(t, map[string]interface{}{
						"id":      "filestream-stream-input-id",
						"enabled": true,
						"type":    "filestream",
						"paths":   logFilePath,
					}),
				},
			},
		},
	}

	healthyOutput := proto.UnitExpected{
		Id:             "output-unit",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_HEALTHY,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "elasticsearch",
			Name: "elasticsearch",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":     "elasticsearch",
					"hosts":    []interface{}{"http://localhost:9200"},
					"username": "admin",
					"password": "testing",
					"protocol": "http",
					"enabled":  true,
				}),
		},
	}

	brokenOutput := proto.UnitExpected{
		Id:             "output-unit-borken",
		Type:           proto.UnitType_OUTPUT,
		ConfigStateIdx: 1,
		State:          proto.State_FAILED,
		LogLevel:       logLevel,
		Config: &proto.UnitExpectedConfig{
			Id:   "default",
			Type: "logstash",
			Name: "logstash",
			Source: integration.RequireNewStruct(t,
				map[string]interface{}{
					"type":    "logstash",
					"invalid": "configuration",
				}),
		},
	}

	// Those are the 'states' Filebeat will go through.
	// After each state is reached the mockServer will
	// send the next.
	protoUnits := [][]*proto.UnitExpected{
		{
			&healthyOutput,
			&filestreamInputHealthy,
		},
		{
			&brokenOutput,
			&filestreamInputStarting,
		},
		{
			&healthyOutput,
			&filestreamInputHealthy,
		},
		{}, // An empty one makes the Beat exit
	}

	// We use `success` to signal the test has ended successfully
	// if `success` is never closed, then the test will fail with a timeout.
	success := make(chan struct{})
	// The test is successful when we reach the last element of `protoUnits`
	onObserved := func(observed *proto.CheckinObserved, protoUnitsIdx int) {
		if protoUnitsIdx == len(protoUnits)-1 {
			close(success)
		}
	}

	server := integration.NewMockServer(
		protoUnits,
		[]uint64{0, 0, 0, 0},
		[]*proto.Features{nil, nil, nil, nil},
		onObserved,
		100*time.Millisecond,
	)
	require.NoError(t, server.Start(), "could not start the mock Elastic-Agent server")
	defer server.Stop()

	filebeat.RestartOnBeatOnExit = true
	filebeat.Start(
		"-E", fmt.Sprintf(`management.insecure_grpc_url_for_testing="localhost:%d"`, server.Port),
		"-E", "management.enabled=true",
		"-E", "management.restart_on_output_change=true",
	)

	select {
	case <-success:
	case <-time.After(60 * time.Second):
		t.Fatal("Output did not recover from a invalid configuration after 60s of waiting")
	}
}

func TestAgentPackageVersion(t *testing.T) {
	want := "8.13.0+build20131123"
	// 1st: mage buildSystemTestBinary
	filebeat := integration.NewBeat(
		t,
		"filebeat",
		"../../filebeat",
	)

	logFilePath := filepath.Join(filebeat.TempDir(), "logs.ndjson")
	generateLogFile(t, logFilePath)

	// output.file:
	//  path: "/tmp/filebeat"
	//  filename: filebeat
	var units = []*proto.UnitExpected{
		{
			Id:             "output-unit",
			Type:           proto.UnitType_OUTPUT,
			ConfigStateIdx: 1,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "default",
				Type: "file",
				Name: "myFile",
				Source: integration.RequireNewStruct(t,
					map[string]interface{}{
						"name": "myLog",
						"type": "file",
						"path": "/tmp/filebeat.ingested.log",
					}),
			},
		},
		{
			Id:             "input-unit-1",
			Type:           proto.UnitType_INPUT,
			ConfigStateIdx: 1,
			State:          proto.State_HEALTHY,
			LogLevel:       proto.UnitLogLevel_DEBUG,
			Config: &proto.UnitExpectedConfig{
				Id:   "filestream-monitoring-agent",
				Type: "filestream",
				Name: "filestream-monitoring-agent",
				Streams: []*proto.Stream{
					{
						Id: "log-input-1",
						Source: integration.RequireNewStruct(t, map[string]interface{}{
							"enabled": true,
							"type":    "log",
							"paths":   []interface{}{logFilePath},
						}),
					},
				},
			},
		},
	}

	server := &mock.StubServerV2{
		CheckinV2Impl: func(observed *proto.CheckinObserved) *proto.CheckinExpected {
			// if management.DoesStateMatch(observed, units[idx], 0) {
			// 	nextState()
			// }
			// for _, unit := range observed.GetUnits() {
			// 	if state := unit.GetState(); !(state != proto.State_CONFIGURING) {
			// 		t.Fatalf("Unit '%s' is not healthy, state: %s", unit.GetId(), unit.GetState().String())
			// 	}
			// }

			// observed.VersionInfo.BuildHash
			return &proto.CheckinExpected{
				Units: units,
			}
		},
		ActionImpl: func(response *proto.ActionResponse) error { return nil },
	}

	priv, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err, "could not create private key")

	notBefore := time.Now()
	notAfter := notBefore.Add(3 * time.Hour)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1653),
		Subject: pkix.Name{
			Organization: []string{"Gallifrey"},
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	ca, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	require.NoError(t, err, "could not create CA")

	privBytes, err := x509.MarshalECPrivateKey(priv)
	require.NoError(t, err, "could not marshal private key")

	var privBytesOut []byte
	privKeyOut := bytes.NewBuffer(privBytesOut)
	err = pem.Encode(privKeyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})
	require.NoError(t, err, "could not pem.Encode private key")

	var certBytesOut []byte
	certOut := bytes.NewBuffer(certBytesOut)
	err = pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: ca})
	require.NoError(t, err, "could not pem.Encode certificate")

	caPEM := certOut.Bytes()
	caTLS, err := tls.X509KeyPair(caPEM, privKeyOut.Bytes())
	require.NoError(t, err, "could not create key pair")

	certPool := x509.NewCertPool()
	ok := certPool.AppendCertsFromPEM(caPEM)
	require.Truef(t, ok, "could not append certs from PEM to cert pool")

	getCert := func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		return &caTLS, nil
	}

	creds := credentials.NewTLS(&tls.Config{
		ClientAuth:     tls.RequireAndVerifyClientCert,
		ClientCAs:      certPool,
		GetCertificate: getCert,
		MinVersion:     tls.VersionTLS12,
	})

	startUpInfo := &proto.StartUpInfo{
		Addr:       fmt.Sprintf("localhost:%d", server.Port),
		ServerName: "mockAgent",
		Token:      "token",
		CaCert:     caPEM,
		PeerCert:   ca,
		PeerKey:    privKeyOut.Bytes(),
		Services:   []proto.ConnInfoServices{proto.ConnInfoServices_CheckinV2},
		AgentInfo: &proto.AgentInfo{
			Id:       uuid.New().String(),
			Version:  want,
			Snapshot: false,
		},
	}

	err = server.Start(grpc.Creds(creds))
	require.NoError(t, err, "failed starting GRPC server")
	t.Cleanup(server.Stop)

	filebeat.Start("-E", "management.enabled=true")

	// With those logs it's possible to see the error reading os.Stdin happens
	// AFTER the file has been written. It seems then the os.Stdin the filebeat.test
	// is reading isn't the file set when it's run.
	// I've also changed to integration.NewBeat to use os/exec.Cmd to check if
	// it could be how the test beat was stated the issue, but it did not change
	// anything.
	// There also is TestStartUpInfo which reads the stdin file and prints out a
	// formatted proto.StartUpInfo.
	// Another option is to use the command `startupinfo` which does the same as
	// TestStartUpInfo. To use it compile filebeat:
	//   - cd x-pack/filebeat
	//   - go build .
	//   - ./filebeat help startupinfo
	//   - ./filebeat startupinfo <build/integration-tests/TestAgentPackageVersionXYZ/stdin
	//
	// My best guess is compiling the test binary interferes with how the process
	// interacts with its stdin. If the test binary is compiled:
	//   - cd x-pack/filebeat
	//   - mage -v buildSystemTestBinary
	//   - ./filebeat.test help
	// you'll see only the flags, not the normal help from filebeat
	t.Logf("[%s] before WriteStartUpInfo", time.Now())
	WriteStartUpInfo(t, filebeat.Stdin(), startUpInfo)
	require.NoError(t, filebeat.Stdin().Sync(), "could not sync beat stdin")
	t.Logf("[%s] after WriteStartUpInfo", time.Now())

	filebeat.WaitForLogs("PublishEvents: ", 10*time.Second, "did not find the logs")
}

func WriteStartUpInfo(t *testing.T, w io.Writer, info *proto.StartUpInfo) {
	t.Helper()
	if len(info.Services) == 0 {
		info.Services = []proto.ConnInfoServices{proto.ConnInfoServices_CheckinV2}
	}

	infoBytes, err := protobuf.Marshal(info)
	require.NoError(t, err, "failed to marshal connection information")

	_, err = w.Write(infoBytes)
	require.NoError(t, err, "failed to write connection information")
}

func TestStartUpInfo(t *testing.T) {
	f, err := os.Open("/home/ainsoph/devel/github.com/elastic/beats/x-pack/filebeat/build/integration-tests/TestAgentPackageVersion2952913677/stdin")
	require.NoError(t, err)

	info, err := client.StartUpInfoFromReader(f)
	require.NoError(t, err)

	bs, err := json.MarshalIndent(info, "", "  ")
	require.NoError(t, err)

	t.Logf("%s", bs)
}

// generateLogFile generates a log file by appending the current
// time to it every second.
func generateLogFile(t *testing.T, fullPath string) {
	t.Helper()
	f, err := os.Create(fullPath)
	if err != nil {
		t.Fatalf("could not create file '%s: %s", fullPath, err)
	}

	go func() {
		t.Helper()
		ticker := time.NewTicker(time.Second)
		t.Cleanup(ticker.Stop)

		done := make(chan struct{})
		t.Cleanup(func() { close(done) })

		defer func() {
			if err := f.Close(); err != nil {
				t.Errorf("could not close log file '%s': %s", fullPath, err)
			}
		}()

		for {
			select {
			case <-done:
				return
			case now := <-ticker.C:
				_, err := fmt.Fprintln(f, now.Format(time.RFC3339))
				if err != nil {
					// The Go compiler does not allow me to call t.Fatalf from a non-test
					// goroutine, so just log it instead
					t.Errorf("could not write data to log file '%s': %s", fullPath, err)
					return
				}
				// make sure log lines are synced as quickly as possible
				if err := f.Sync(); err != nil {
					t.Errorf("could not sync file '%s': %s", fullPath, err)
				}
			}
		}
	}()
}
