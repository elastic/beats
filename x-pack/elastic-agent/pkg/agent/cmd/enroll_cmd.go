// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"

	"github.com/elastic/beats/v7/libbeat/common/transport/httpcommon"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/filelock"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configuration"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/control/proto"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/authority"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	fleetclient "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/remote"
)

const (
	maxRetriesstoreAgentInfo = 5
	waitingForAgent          = "Waiting for Elastic Agent to start"
	waitingForFleetServer    = "Waiting for Elastic Agent to start Fleet Server"
	defaultFleetServerHost   = "0.0.0.0"
	defaultFleetServerPort   = 8220
)

var (
	enrollDelay   = 1 * time.Second  // max delay to start enrollment
	daemonTimeout = 30 * time.Second // max amount of for communication to running Agent daemon
)

type saver interface {
	Save(io.Reader) error
}

// enrollCmd is an enroll subcommand that interacts between the Kibana API and the Agent.
type enrollCmd struct {
	log          *logger.Logger
	options      *enrollCmdOption
	client       fleetclient.Sender
	configStore  saver
	remoteConfig remote.Config
	agentProc    *process.Info
	configPath   string
}

// enrollCmdFleetServerOption define all the supported enrollment options for bootstrapping with Fleet Server.
type enrollCmdFleetServerOption struct {
	ConnStr         string
	ElasticsearchCA string
	ServiceToken    string
	PolicyID        string
	Host            string
	Port            uint16
	Cert            string
	CertKey         string
	Insecure        bool
	SpawnAgent      bool
	Headers         map[string]string
	ProxyURL        string
	ProxyDisabled   bool
	ProxyHeaders    map[string]string
}

// enrollCmdOption define all the supported enrollment option.
type enrollCmdOption struct {
	ID                   string
	URL                  string
	CAs                  []string
	CASha256             []string
	Insecure             bool
	UserProvidedMetadata map[string]interface{}
	EnrollAPIKey         string
	Staging              string
	FleetServer          enrollCmdFleetServerOption
}

func (e *enrollCmdOption) remoteConfig() (remote.Config, error) {
	cfg, err := remote.NewConfigFromURL(e.URL)
	if err != nil {
		return remote.Config{}, err
	}
	if cfg.Protocol == remote.ProtocolHTTP && !e.Insecure {
		return remote.Config{}, fmt.Errorf("connection to fleet-server is insecure, strongly recommended to use a secure connection (override with --insecure)")
	}

	var tlsCfg tlscommon.Config

	// Add any SSL options from the CLI.
	if len(e.CAs) > 0 || len(e.CASha256) > 0 {
		tlsCfg.CAs = e.CAs
		tlsCfg.CASha256 = e.CASha256
	}
	if e.Insecure {
		tlsCfg.VerificationMode = tlscommon.VerifyNone
	}

	cfg.Transport.TLS = &tlsCfg

	var proxyURL *url.URL
	if e.FleetServer.ProxyURL != "" {
		proxyURL, err = common.ParseURL(e.FleetServer.ProxyURL)
		if err != nil {
			return remote.Config{}, err
		}
	}

	var headers http.Header
	if len(e.FleetServer.ProxyHeaders) > 0 {
		headers = http.Header{}
		for k, v := range e.FleetServer.ProxyHeaders {
			headers.Add(k, v)
		}
	}

	cfg.Transport.Proxy = httpcommon.HTTPClientProxySettings{
		URL:     proxyURL,
		Disable: e.FleetServer.ProxyDisabled,
		Headers: headers,
	}

	return cfg, nil
}

// newEnrollCmd creates a new enroll command that will registers the current beats to the remote
// system.
func newEnrollCmd(
	log *logger.Logger,
	options *enrollCmdOption,
	configPath string,
) (*enrollCmd, error) {

	store := storage.NewReplaceOnSuccessStore(
		configPath,
		application.DefaultAgentFleetConfig,
		storage.NewDiskStore(paths.AgentConfigFile()),
	)

	return newEnrollCmdWithStore(
		log,
		options,
		configPath,
		store,
	)
}

// newEnrollCmdWithStore creates an new enrollment and accept a custom store.
func newEnrollCmdWithStore(
	log *logger.Logger,
	options *enrollCmdOption,
	configPath string,
	store saver,
) (*enrollCmd, error) {
	return &enrollCmd{
		log:         log,
		options:     options,
		configStore: store,
		configPath:  configPath,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *enrollCmd) Execute(ctx context.Context) error {
	var err error
	defer c.stopAgent() // ensure its stopped no matter what

	persistentConfig, err := getPersistentConfig(c.configPath)
	if err != nil {
		return err
	}

	// localFleetServer indicates that we start our internal fleet server. Agent
	// will communicate to the internal fleet server on localhost only.
	// Connection setup should disable proxies in that case.
	localFleetServer := c.options.FleetServer.ConnStr != ""
	if localFleetServer {
		token, err := c.fleetServerBootstrap(ctx)
		if err != nil {
			return err
		}
		if c.options.EnrollAPIKey == "" && token != "" {
			c.options.EnrollAPIKey = token
		}
	}

	c.remoteConfig, err = c.options.remoteConfig()
	if err != nil {
		return errors.New(
			err, "Error",
			errors.TypeConfig,
			errors.M(errors.MetaKeyURI, c.options.URL))
	}
	if localFleetServer {
		// Ensure that the agent does not use a proxy configuration
		// when connecting to the local fleet server.
		c.remoteConfig.Transport.Proxy.Disable = true
	}

	c.client, err = fleetclient.NewWithConfig(c.log, c.remoteConfig)
	if err != nil {
		return errors.New(
			err, "Error",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, c.options.URL))
	}

	err = c.enrollWithBackoff(ctx, persistentConfig)
	if err != nil {
		return errors.New(err, "fail to enroll")
	}

	if c.agentProc == nil {
		if c.daemonReload(ctx) != nil {
			c.log.Info("Elastic Agent might not be running; unable to trigger restart")
		}
		c.log.Info("Successfully triggered restart on running Elastic Agent.")
		return nil
	}
	c.log.Info("Elastic Agent has been enrolled; start Elastic Agent")
	return nil
}

func (c *enrollCmd) fleetServerBootstrap(ctx context.Context) (string, error) {
	c.log.Debug("verifying communication with running Elastic Agent daemon")
	agentRunning := true
	_, err := getDaemonStatus(ctx)
	if err != nil {
		if !c.options.FleetServer.SpawnAgent {
			// wait longer to try and communicate with the Elastic Agent
			err = waitForAgent(ctx)
			if err != nil {
				return "", errors.New("failed to communicate with elastic-agent daemon; is elastic-agent running?")
			}
		} else {
			agentRunning = false
		}
	}

	err = c.prepareFleetTLS()
	if err != nil {
		return "", err
	}

	fleetConfig, err := createFleetServerBootstrapConfig(
		c.options.FleetServer.ConnStr, c.options.FleetServer.ServiceToken,
		c.options.FleetServer.PolicyID,
		c.options.FleetServer.Host, c.options.FleetServer.Port,
		c.options.FleetServer.Cert, c.options.FleetServer.CertKey, c.options.FleetServer.ElasticsearchCA,
		c.options.FleetServer.Headers,
		c.options.FleetServer.ProxyURL,
		c.options.FleetServer.ProxyDisabled,
		c.options.FleetServer.ProxyHeaders,
	)
	if err != nil {
		return "", err
	}

	configToStore := map[string]interface{}{
		"fleet": fleetConfig,
	}
	reader, err := yamlToReader(configToStore)
	if err != nil {
		return "", err
	}

	if err := safelyStoreAgentInfo(c.configStore, reader); err != nil {
		return "", err
	}

	var agentSubproc <-chan *os.ProcessState
	if agentRunning {
		// reload the already running agent
		err = c.daemonReloadWithBackoff(ctx)
		if err != nil {
			return "", errors.New(err, "failed to trigger elastic-agent daemon reload", errors.TypeApplication)
		}
	} else {
		// spawn `run` as a subprocess so enroll can perform the bootstrap process of Fleet Server
		agentSubproc, err = c.startAgent(ctx)
		if err != nil {
			return "", err
		}
	}

	token, err := waitForFleetServer(ctx, agentSubproc, c.log)
	if err != nil {
		return "", errors.New(err, "fleet-server never started by elastic-agent daemon", errors.TypeApplication)
	}
	return token, nil
}

func (c *enrollCmd) prepareFleetTLS() error {
	host := c.options.FleetServer.Host
	if host == "" {
		host = "localhost"
	}
	port := c.options.FleetServer.Port
	if port == 0 {
		port = defaultFleetServerPort
	}
	if c.options.FleetServer.Cert != "" && c.options.FleetServer.CertKey == "" {
		return errors.New("certificate private key is required when certificate provided")
	}
	if c.options.FleetServer.CertKey != "" && c.options.FleetServer.Cert == "" {
		return errors.New("certificate is required when certificate private key is provided")
	}
	if c.options.FleetServer.Cert == "" && c.options.FleetServer.CertKey == "" {
		if c.options.FleetServer.Insecure {
			// running insecure, force the binding to localhost (unless specified)
			if c.options.FleetServer.Host == "" {
				c.options.FleetServer.Host = "localhost"
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
		c.options.FleetServer.Cert = string(pair.Crt)
		c.options.FleetServer.CertKey = string(pair.Key)
		c.options.URL = fmt.Sprintf("https://%s:%d", hostname, port)
		c.options.CAs = []string{string(ca.Crt())}
	}
	// running with custom Cert and CertKey; URL is required to be set
	if c.options.URL == "" {
		return errors.New("url is required when a certificate is provided")
	}
	return nil
}

func (c *enrollCmd) daemonReloadWithBackoff(ctx context.Context) error {
	err := c.daemonReload(ctx)
	if err == nil {
		return nil
	}

	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, 10*time.Second, 1*time.Minute)

	for i := 5; i >= 0; i-- {
		backExp.Wait()
		c.log.Info("Retrying to restart...")
		err = c.daemonReload(ctx)
		if err == nil {
			break
		}
	}

	close(signal)
	return err
}

func (c *enrollCmd) daemonReload(ctx context.Context) error {
	daemon := client.New()
	err := daemon.Connect(ctx)
	if err != nil {
		return err
	}
	defer daemon.Disconnect()
	return daemon.Restart(ctx)
}

func (c *enrollCmd) enrollWithBackoff(ctx context.Context, persistentConfig map[string]interface{}) error {
	delay(ctx, enrollDelay)

	c.log.Infof("Starting enrollment to URL: %s", c.client.URI())
	err := c.enroll(ctx, persistentConfig)
	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, 60*time.Second, 10*time.Minute)

	for {
		retry := false
		if errors.Is(err, fleetapi.ErrTooManyRequests) {
			c.log.Warn("Too many requests on the remote server, will retry in a moment.")
			retry = true
		} else if errors.Is(err, fleetapi.ErrConnRefused) {
			c.log.Warn("Remote server is not ready to accept connections, will retry in a moment.")
			retry = true
		}
		if !retry {
			break
		}
		backExp.Wait()
		c.log.Infof("Retrying enrollment to URL: %s", c.client.URI())
		err = c.enroll(ctx, persistentConfig)
	}

	close(signal)
	return err
}

func (c *enrollCmd) enroll(ctx context.Context, persistentConfig map[string]interface{}) error {
	cmd := fleetapi.NewEnrollCmd(c.client)

	metadata, err := info.Metadata()
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
			"fail to execute request to fleet-server",
			errors.TypeNetwork)
	}

	fleetConfig, err := createFleetConfigFromEnroll(resp.Item.AccessAPIKey, c.remoteConfig)
	if err != nil {
		return err
	}

	agentConfig, err := c.createAgentConfig(resp.Item.ID, persistentConfig, c.options.FleetServer.Headers)
	if err != nil {
		return err
	}

	localFleetServer := c.options.FleetServer.ConnStr != ""
	if localFleetServer {
		serverConfig, err := createFleetServerBootstrapConfig(
			c.options.FleetServer.ConnStr, c.options.FleetServer.ServiceToken,
			c.options.FleetServer.PolicyID,
			c.options.FleetServer.Host, c.options.FleetServer.Port,
			c.options.FleetServer.Cert, c.options.FleetServer.CertKey, c.options.FleetServer.ElasticsearchCA,
			c.options.FleetServer.Headers,
			c.options.FleetServer.ProxyURL, c.options.FleetServer.ProxyDisabled, c.options.FleetServer.ProxyHeaders)
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

	if err := safelyStoreAgentInfo(c.configStore, reader); err != nil {
		return err
	}

	// clear action store
	// fail only if file exists and there was a failure
	if err := os.Remove(paths.AgentActionStoreFile()); !os.IsNotExist(err) {
		return err
	}

	// clear action store
	// fail only if file exists and there was a failure
	if err := os.Remove(paths.AgentStateStoreFile()); !os.IsNotExist(err) {
		return err
	}

	return nil
}

func (c *enrollCmd) startAgent(ctx context.Context) (<-chan *os.ProcessState, error) {
	cmd, err := os.Executable()
	if err != nil {
		return nil, err
	}
	c.log.Info("Spawning Elastic Agent daemon as a subprocess to complete bootstrap process.")
	args := []string{
		"run", "-e", "-c", paths.ConfigFile(),
		"--path.home", paths.Top(), "--path.config", paths.Config(),
		"--path.logs", paths.Logs(),
	}
	if !paths.IsVersionHome() {
		args = append(args, "--path.home.unversioned")
	}
	proc, err := process.StartContext(
		ctx, c.log, cmd, nil, os.Geteuid(), os.Getegid(), args, func(c *exec.Cmd) {
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
		})
	if err != nil {
		return nil, err
	}
	resChan := make(chan *os.ProcessState)
	go func() {
		procState, _ := proc.Process.Wait()
		resChan <- procState
	}()
	c.agentProc = proc
	return resChan, nil
}

func (c *enrollCmd) stopAgent() {
	if c.agentProc != nil {
		c.agentProc.StopWait()
		c.agentProc = nil
	}
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
	enrollmentToken string
	err             error
}

func waitForAgent(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	resChan := make(chan waitResult)
	innerCtx, innerCancel := context.WithCancel(context.Background())
	defer innerCancel()
	go func() {
		for {
			<-time.After(1 * time.Second)
			_, err := getDaemonStatus(innerCtx)
			if err == context.Canceled {
				resChan <- waitResult{err: err}
				return
			}
			if err == nil {
				resChan <- waitResult{}
				break
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

func waitForFleetServer(ctx context.Context, agentSubproc <-chan *os.ProcessState, log *logger.Logger) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	resChan := make(chan waitResult)
	innerCtx, innerCancel := context.WithCancel(context.Background())
	defer innerCancel()
	go func() {
		msg := ""
		msgCount := 0
		for {
			<-time.After(1 * time.Second)
			status, err := getDaemonStatus(innerCtx)
			if err == context.Canceled {
				resChan <- waitResult{err: err}
				return
			}
			if err != nil {
				log.Debugf("%s: %s", waitingForAgent, err)
				if msg != waitingForAgent {
					msg = waitingForAgent
					msgCount = 0
					log.Info(waitingForAgent)
				} else {
					msgCount++
					if msgCount > 5 {
						msgCount = 0
						log.Infof("%s: %s", waitingForAgent, err)
					}
				}
				continue
			}
			app := getAppFromStatus(status, "fleet-server")
			if app == nil {
				err = errors.New("no fleet-server application running")
				log.Debugf("%s: %s", waitingForFleetServer, err)
				if msg != waitingForFleetServer {
					msg = waitingForFleetServer
					msgCount = 0
					log.Info(waitingForFleetServer)
				} else {
					msgCount++
					if msgCount > 5 {
						msgCount = 0
						log.Infof("%s: %s", waitingForFleetServer, err)
					}
				}
				continue
			}
			log.Debugf("%s: %s - %s", waitingForFleetServer, app.Status, app.Message)
			if app.Status == proto.Status_DEGRADED || app.Status == proto.Status_HEALTHY {
				// app has started and is running
				if app.Message != "" {
					log.Infof("Fleet Server - %s", app.Message)
				}
				// extract the enrollment token from the status payload
				token := ""
				if app.Payload != nil {
					if enrollToken, ok := app.Payload["enrollment_token"]; ok {
						if tokenStr, ok := enrollToken.(string); ok {
							token = tokenStr
						}
					}
				}
				resChan <- waitResult{enrollmentToken: token}
				break
			}
			if app.Message != "" {
				appMsg := fmt.Sprintf("Fleet Server - %s", app.Message)
				if msg != appMsg {
					msg = appMsg
					msgCount = 0
					log.Info(appMsg)
				} else {
					msgCount++
					if msgCount > 5 {
						msgCount = 0
						log.Info(appMsg)
					}
				}
			}
		}
	}()

	var res waitResult
	if agentSubproc == nil {
		select {
		case <-ctx.Done():
			innerCancel()
			res = <-resChan
		case res = <-resChan:
		}
	} else {
		select {
		case ps := <-agentSubproc:
			res = waitResult{err: fmt.Errorf("spawned Elastic Agent exited unexpectedly: %s", ps)}
		case <-ctx.Done():
			innerCancel()
			res = <-resChan
		case res = <-resChan:
		}
	}

	if res.err != nil {
		return "", res.err
	}
	return res.enrollmentToken, nil
}

func getAppFromStatus(status *client.AgentStatus, name string) *client.ApplicationStatus {
	for _, app := range status.Applications {
		if app.Name == name {
			return app
		}
	}
	return nil
}

func safelyStoreAgentInfo(s saver, reader io.Reader) error {
	var err error
	signal := make(chan struct{})
	backExp := backoff.NewExpBackoff(signal, 100*time.Millisecond, 3*time.Second)

	for i := 0; i <= maxRetriesstoreAgentInfo; i++ {
		backExp.Wait()
		err = storeAgentInfo(s, reader)
		if err != filelock.ErrAppAlreadyRunning {
			break
		}
	}

	close(signal)
	return err
}

func storeAgentInfo(s saver, reader io.Reader) error {
	fileLock := paths.AgentConfigFileLock()
	if err := fileLock.TryLock(); err != nil {
		return err
	}
	defer fileLock.Unlock()

	if err := s.Save(reader); err != nil {
		return errors.New(err, "could not save enrollment information", errors.TypeFilesystem)
	}

	return nil
}

func createFleetServerBootstrapConfig(
	connStr, serviceToken, policyID, host string,
	port uint16,
	cert, key, esCA string,
	headers map[string]string,
	proxyURL string,
	proxyDisabled bool,
	proxyHeaders map[string]string,
) (*configuration.FleetAgentConfig, error) {
	localFleetServer := connStr != ""

	es, err := configuration.ElasticsearchFromConnStr(connStr, serviceToken)
	if err != nil {
		return nil, err
	}
	if esCA != "" {
		es.TLS = &tlscommon.Config{
			CAs: []string{esCA},
		}
	}
	if host == "" {
		host = defaultFleetServerHost
	}
	if port == 0 {
		port = defaultFleetServerPort
	}
	if len(headers) > 0 {
		if es.Headers == nil {
			es.Headers = make(map[string]string)
		}
		// overwrites previously set headers
		for k, v := range headers {
			es.Headers[k] = v
		}
	}
	es.ProxyURL = proxyURL
	es.ProxyDisabled = proxyDisabled
	es.ProxyHeaders = proxyHeaders

	cfg := configuration.DefaultFleetAgentConfig()
	cfg.Enabled = true
	cfg.Server = &configuration.FleetServerConfig{
		Bootstrap: true,
		Output: configuration.FleetServerOutputConfig{
			Elasticsearch: es,
		},
		Host: host,
		Port: port,
	}
	if policyID != "" {
		cfg.Server.Policy = &configuration.FleetServerPolicyConfig{ID: policyID}
	}
	if cert != "" || key != "" {
		cfg.Server.TLS = &tlscommon.Config{
			Certificate: tlscommon.CertificateConfig{
				Certificate: cert,
				Key:         key,
			},
		}
	}

	if localFleetServer {
		cfg.Client.Transport.Proxy.Disable = true
	}

	if err := cfg.Valid(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}

func createFleetConfigFromEnroll(accessAPIKey string, cli remote.Config) (*configuration.FleetAgentConfig, error) {
	cfg := configuration.DefaultFleetAgentConfig()
	cfg.Enabled = true
	cfg.AccessAPIKey = accessAPIKey
	cfg.Client = cli

	if err := cfg.Valid(); err != nil {
		return nil, errors.New(err, "invalid enrollment options", errors.TypeConfig)
	}
	return cfg, nil
}

func (c *enrollCmd) createAgentConfig(agentID string, pc map[string]interface{}, headers map[string]string) (map[string]interface{}, error) {
	agentConfig := map[string]interface{}{
		"id": agentID,
	}

	if len(headers) > 0 {
		agentConfig["headers"] = headers
	}

	if c.options.Staging != "" {
		staging := fmt.Sprintf("https://staging.elastic.co/%s-%s/downloads/", release.Version(), c.options.Staging[:8])
		agentConfig["download"] = map[string]interface{}{
			"sourceURI": staging,
		}
	}

	for k, v := range pc {
		agentConfig[k] = v
	}

	return agentConfig, nil
}

func getPersistentConfig(pathConfigFile string) (map[string]interface{}, error) {
	persistentMap := make(map[string]interface{})
	rawConfig, err := config.LoadFile(pathConfigFile)
	if os.IsNotExist(err) {
		return persistentMap, nil
	}
	if err != nil {
		return nil, errors.New(err,
			fmt.Sprintf("could not read configuration file %s", pathConfigFile),
			errors.TypeFilesystem,
			errors.M(errors.MetaKeyPath, pathConfigFile))
	}

	pc := &struct {
		LogLevel       string                                 `json:"agent.logging.level,omitempty" yaml:"agent.logging.level,omitempty" config:"agent.logging.level,omitempty"`
		MonitoringHTTP *monitoringConfig.MonitoringHTTPConfig `json:"agent.monitoring.http,omitempty" yaml:"agent.monitoring.http,omitempty" config:"agent.monitoring.http,omitempty"`
	}{
		MonitoringHTTP: monitoringConfig.DefaultConfig().HTTP,
	}

	if err := rawConfig.Unpack(&pc); err != nil {
		return nil, err
	}

	if pc.LogLevel != "" {
		persistentMap["logging.level"] = pc.LogLevel
	}

	if pc.MonitoringHTTP != nil {
		persistentMap["monitoring.http"] = pc.MonitoringHTTP
	}

	return persistentMap, nil
}
