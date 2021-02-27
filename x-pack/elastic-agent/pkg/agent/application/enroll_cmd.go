// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/backoff"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/authority"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

const (
	waitingForAgent        = "waiting for Elastic Agent to start"
	waitingForFleetServer  = "waiting for Elastic Agent to start Fleet Server"
	defaultFleetServerPort = 8220
)

var (
	enrollDelay   = 1 * time.Second  // max delay to start enrollment
	daemonTimeout = 30 * time.Second // max amount of for communication to running Agent daemon
)

type store interface {
	Save(io.Reader) error
}

type storeLoad interface {
	store
	Load() (io.ReadCloser, error)
}

type clienter interface {
	Send(
		ctx context.Context,
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)

	URI() string
}

// EnrollCmd is an enroll subcommand that interacts between the Kibana API and the Agent.
type EnrollCmd struct {
	log          *logger.Logger
	options      *EnrollCmdOption
	client       clienter
	configStore  store
	kibanaConfig *kibana.Config
}

// EnrollCmdOption define all the supported enrollment option.
type EnrollCmdOption struct {
	ID                   string
	URL                  string
	CAs                  []string
	CASha256             []string
	Insecure             bool
	UserProvidedMetadata map[string]interface{}
	EnrollAPIKey         string
	Staging              string
	FleetServerConnStr   string
	FleetServerPolicyID  string
	FleetServerHost      string
	FleetServerPort      uint16
	FleetServerCert      string
	FleetServerCertKey   string
	FleetServerInsecure  bool
}

func (e *EnrollCmdOption) kibanaConfig() (*kibana.Config, error) {
	cfg, err := kibana.NewConfigFromURL(e.URL)
	if err != nil {
		return nil, err
	}
	if cfg.Protocol == kibana.ProtocolHTTP && !e.Insecure {
		return nil, fmt.Errorf("connection to Kibana is insecure, strongly recommended to use a secure connection (override with --insecure)")
	}

	// Add any SSL options from the CLI.
	if len(e.CAs) > 0 || len(e.CASha256) > 0 {
		cfg.TLS = &tlscommon.Config{
			CAs:      e.CAs,
			CASha256: e.CASha256,
		}
	}
	if e.Insecure {
		cfg.TLS = &tlscommon.Config{
			VerificationMode: tlscommon.VerifyNone,
		}
	}

	return cfg, nil
}

// NewEnrollCmd creates a new enroll command that will registers the current beats to the remote
// system.
func NewEnrollCmd(
	log *logger.Logger,
	options *EnrollCmdOption,
	configPath string,
) (*EnrollCmd, error) {

	store := storage.NewReplaceOnSuccessStore(
		configPath,
		DefaultAgentFleetConfig,
		storage.NewDiskStore(info.AgentConfigFile()),
	)

	return NewEnrollCmdWithStore(
		log,
		options,
		configPath,
		store,
	)
}

//NewEnrollCmdWithStore creates an new enrollment and accept a custom store.
func NewEnrollCmdWithStore(
	log *logger.Logger,
	options *EnrollCmdOption,
	configPath string,
	store store,
) (*EnrollCmd, error) {
	return &EnrollCmd{
		log:         log,
		options:     options,
		configStore: store,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *EnrollCmd) Execute(ctx context.Context) error {
	var err error
	if c.options.FleetServerConnStr != "" {
		err = c.fleetServerBootstrap(ctx)
		if err != nil {
			return err
		}
	}

	c.kibanaConfig, err = c.options.kibanaConfig()
	if err != nil {
		return errors.New(
			err, "Error",
			errors.TypeConfig,
			errors.M(errors.MetaKeyURI, c.options.URL))
	}

	c.client, err = fleetapi.NewWithConfig(c.log, c.kibanaConfig)
	if err != nil {
		return errors.New(
			err, "Error",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, c.options.URL))
	}

	err = c.enrollWithBackoff(ctx)
	if err != nil {
		return errors.New(err, "fail to enroll")
	}

	if c.daemonReload(ctx) != nil {
		c.log.Info("Elastic Agent might not be running; unable to trigger restart")
	}
	c.log.Info("Successfully triggered restart on running Elastic Agent.")
	return nil
}

func (c *EnrollCmd) fleetServerBootstrap(ctx context.Context) error {
	c.log.Debug("verifying communication with running Elastic Agent daemon")
	_, err := getDaemonStatus(ctx)
	if err != nil {
		return errors.New("failed to communicate with elastic-agent daemon; is elastic-agent running?")
	}

	err = c.prepareFleetTLS()
	if err != nil {
		return err
	}

	fleetConfig, err := createFleetServerBootstrapConfig(
		c.options.FleetServerConnStr, c.options.FleetServerPolicyID,
		c.options.FleetServerHost, c.options.FleetServerPort,
		c.options.FleetServerCert, c.options.FleetServerCertKey)
	configToStore := map[string]interface{}{
		"fleet": fleetConfig,
	}
	reader, err := yamlToReader(configToStore)
	if err != nil {
		return err
	}
	if err := c.configStore.Save(reader); err != nil {
		return errors.New(err, "could not save fleet server bootstrap information", errors.TypeFilesystem)
	}

	err = c.daemonReload(ctx)
	if err != nil {
		return errors.New(err, "failed to trigger elastic-agent daemon reload", errors.TypeApplication)
	}

	err = waitForFleetServer(ctx, c.log)
	if err != nil {
		return errors.New(err, "fleet-server never started by elastic-agent daemon", errors.TypeApplication)
	}
	return nil
}

func (c *EnrollCmd) prepareFleetTLS() error {
	host := c.options.FleetServerHost
	if host == "" {
		host = "localhost"
	}
	port := c.options.FleetServerPort
	if port == 0 {
		port = defaultFleetServerPort
	}
	if c.options.FleetServerCert != "" && c.options.FleetServerCertKey == "" {
		return errors.New("certificate private key is required when certificate provided")
	}
	if c.options.FleetServerCertKey != "" && c.options.FleetServerCert == "" {
		return errors.New("certificate is required when certificate private key is provided")
	}
	if c.options.FleetServerCert == "" && c.options.FleetServerCertKey == "" {
		if c.options.FleetServerInsecure {
			// running insecure, force the binding to localhost (unless specified)
			if c.options.FleetServerHost == "" {
				c.options.FleetServerHost = "localhost"
			}
			c.options.URL = fmt.Sprintf("http://%s:%d", host, port)
			c.options.Insecure = true
			return nil
		}

		c.log.Info("Generating self-signed certificate for Fleet Server")
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		ca, err := authority.NewCA()
		if err != nil {
			return err
		}
		pair, err := ca.GeneratePairWithName(hostname)
		if err != nil {
			return err
		}
		c.options.FleetServerCert = string(pair.Crt)
		c.options.FleetServerCertKey = string(pair.Key)
		c.options.URL = fmt.Sprintf("https://%s:%d", hostname, port)
		c.options.CAs = []string{string(ca.Crt())}
	}
	return nil
}

func (c *EnrollCmd) daemonReload(ctx context.Context) error {
	daemon := client.New()
	err := daemon.Connect(ctx)
	if err != nil {
		return err
	}
	defer daemon.Disconnect()
	return daemon.Restart(ctx)
}

func (c *EnrollCmd) enrollWithBackoff(ctx context.Context) error {
	delay(ctx, enrollDelay)

	err := c.enroll(ctx)
	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, 60*time.Second, 10*time.Minute)

	for errors.Is(err, fleetapi.ErrTooManyRequests) {
		c.log.Warn("Too many requests on the remote server, will retry in a moment.")
		backExp.Wait()
		c.log.Info("Retrying to enroll...")
		err = c.enroll(ctx)
	}

	close(signal)
	return err
}

func (c *EnrollCmd) enroll(ctx context.Context) error {
	cmd := fleetapi.NewEnrollCmd(c.client)

	metadata, err := metadata()
	if err != nil {
		return errors.New(err, "acquiring metadata failed")
	}

	r := &fleetapi.EnrollRequest{
		EnrollAPIKey: c.options.EnrollAPIKey,
		SharedID:     c.options.ID,
		Type:         fleetapi.PermanentEnroll,
		Metadata: fleetapi.Metadata{
			Local:        metadata,
			UserProvided: c.options.UserProvidedMetadata,
		},
	}

	resp, err := cmd.Execute(ctx, r)
	if err != nil {
		return errors.New(err,
			"fail to execute request to Kibana",
			errors.TypeNetwork)
	}

	fleetConfig, err := createFleetConfigFromEnroll(resp.Item.AccessAPIKey, c.kibanaConfig)
	if err != nil {
		return err
	}
	agentConfig := map[string]interface{}{
		"id": resp.Item.ID,
	}
	if c.options.Staging != "" {
		staging := fmt.Sprintf("https://staging.elastic.co/%s-%s/downloads/", release.Version(), c.options.Staging[:8])
		agentConfig["download"] = map[string]interface{}{
			"sourceURI": staging,
		}
	}
	if c.options.FleetServerConnStr != "" {
		serverConfig, err := createFleetServerBootstrapConfig(
			c.options.FleetServerConnStr, c.options.FleetServerPolicyID,
			c.options.FleetServerHost, c.options.FleetServerPort,
			c.options.FleetServerCert, c.options.FleetServerCertKey)
		if err != nil {
			return err
		}
		// no longer need bootstrap at this point
		serverConfig.Server.Bootstrap = false
		fleetConfig.Server = serverConfig.Server
	}

	configToStore := map[string]interface{}{
		"fleet": fleetConfig,
		"agent": agentConfig,
	}

	reader, err := yamlToReader(configToStore)
	if err != nil {
		return err
	}

	if err := c.configStore.Save(reader); err != nil {
		return errors.New(err, "could not save enrollment information", errors.TypeFilesystem)
	}

	if _, err := info.NewAgentInfo(); err != nil {
		return err
	}

	// clear action store
	// fail only if file exists and there was a failure
	if err := os.Remove(info.AgentActionStoreFile()); !os.IsNotExist(err) {
		return err
	}

	// clear action store
	// fail only if file exists and there was a failure
	if err := os.Remove(info.AgentStateStoreFile()); !os.IsNotExist(err) {
		return err
	}

	return nil
}

func yamlToReader(in interface{}) (io.Reader, error) {
	data, err := yaml.Marshal(in)
	if err != nil {
		return nil, errors.New(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}

func delay(ctx context.Context, d time.Duration) {
	t := time.NewTimer(time.Duration(rand.Int63n(int64(d))))
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

func getDaemonStatus(ctx context.Context) (*client.AgentStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, daemonTimeout)
	defer cancel()
	daemon := client.New()
	err := daemon.Connect(ctx)
	if err != nil {
		return nil, err
	}
	defer daemon.Disconnect()
	return daemon.Status(ctx)
}

type waitResult struct {
	err error
}

func waitForFleetServer(ctx context.Context, log *logger.Logger) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	resChan := make(chan waitResult)
	innerCtx, innerCancel := context.WithCancel(context.Background())
	defer innerCancel()
	go func() {
		msg := ""
		for {
			<-time.After(1 * time.Second)
			status, err := getDaemonStatus(innerCtx)
			if err == context.Canceled {
				resChan <- waitResult{err: err}
				return
			}
			if err != nil {
				log.Debug(waitingForAgent)
				if msg != waitingForAgent {
					msg = waitingForAgent
					log.Info(waitingForAgent)
				}
				continue
			}
			app := getAppFromStatus(status, "fleet-server")
			if app == nil {
				log.Debug(waitingForFleetServer)
				if msg != waitingForFleetServer {
					msg = waitingForFleetServer
					log.Info(waitingForFleetServer)
				}
				continue
			}
			log.Debugf("fleet-server status: %s - %s", app.Status, app.Message)
			if app.Status == proto.Status_DEGRADED || app.Status == proto.Status_HEALTHY {
				// app has started and is running
				resChan <- waitResult{}
				break
			}
			if app.Message != "" {
				appMsg := fmt.Sprintf("Fleet Server - %s", app.Message)
				if msg != appMsg {
					msg = appMsg
					log.Info(appMsg)
				}
			}
		}
	}()

	var res waitResult
	select {
	case <-ctx.Done():
		innerCancel()
		res = <-resChan
	case res = <-resChan:
	}

	if res.err != nil {
		return res.err
	}
	return nil
}

func getAppFromStatus(status *client.AgentStatus, name string) *client.ApplicationStatus {
	for _, app := range status.Applications {
		if app.Name == name {
			return app
		}
	}
	return nil
}
