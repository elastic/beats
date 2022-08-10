// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"time"

	"gopkg.in/yaml.v2"

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
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/install"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/cli"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/authority"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/backoff"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	monitoringConfig "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/process"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	fleetclient "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi/client"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/remote"
)

const (
	maxRetriesstoreAgentInfo       = 5
	waitingForAgent                = "Waiting for Elastic Agent to start"
	waitingForFleetServer          = "Waiting for Elastic Agent to start Fleet Server"
	defaultFleetServerHost         = "0.0.0.0"
	defaultFleetServerPort         = 8220
	defaultFleetServerInternalHost = "localhost"
	defaultFleetServerInternalPort = 8221
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
	ConnStr               string
	ElasticsearchCA       string
	ElasticsearchCASHA256 string
	ElasticsearchInsecure bool
	ServiceToken          string
	PolicyID              string
	Host                  string
	Port                  uint16
	InternalPort          uint16
	Cert                  string
	CertKey               string
	Insecure              bool
	SpawnAgent            bool
	Headers               map[string]string
	Timeout               time.Duration
}

// enrollCmdOption define all the supported enrollment option.
type enrollCmdOption struct {
	URL                  string                     `yaml:"url,omitempty"`
	InternalURL          string                     `yaml:"-"`
	CAs                  []string                   `yaml:"ca,omitempty"`
	CASha256             []string                   `yaml:"ca_sha256,omitempty"`
	Insecure             bool                       `yaml:"insecure,omitempty"`
	EnrollAPIKey         string                     `yaml:"enrollment_key,omitempty"`
	Staging              string                     `yaml:"staging,omitempty"`
	ProxyURL             string                     `yaml:"proxy_url,omitempty"`
	ProxyDisabled        bool                       `yaml:"proxy_disabled,omitempty"`
	ProxyHeaders         map[string]string          `yaml:"proxy_headers,omitempty"`
	DaemonTimeout        time.Duration              `yaml:"daemon_timeout,omitempty"`
	UserProvidedMetadata map[string]interface{}     `yaml:"-"`
	FixPermissions       bool                       `yaml:"-"`
	DelayEnroll          bool                       `yaml:"-"`
	FleetServer          enrollCmdFleetServerOption `yaml:"-"`
}

// remoteConfig returns the configuration used to connect the agent to a fleet process.
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

	proxySettings, err := httpcommon.NewHTTPClientProxySettings(e.ProxyURL, e.ProxyHeaders, e.ProxyDisabled)
	if err != nil {
		return remote.Config{}, err
	}

	cfg.Transport.Proxy = *proxySettings

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
func (c *enrollCmd) Execute(ctx context.Context, streams *cli.IOStreams) error {
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
	if localFleetServer && !c.options.DelayEnroll {
		token, err := c.fleetServerBootstrap(ctx, persistentConfig)
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

	if c.options.DelayEnroll {
		if c.options.FleetServer.Host != "" {
			return errors.New("--delay-enroll cannot be used with --fleet-server-es", errors.TypeConfig)
		}
		return c.writeDelayEnroll(streams)
	}

	err = c.enrollWithBackoff(ctx, persistentConfig)
	if err != nil {
		return errors.New(err, "fail to enroll")
	}

	if c.options.FixPermissions {
		err = install.FixPermissions()
		if err != nil {
			return errors.New(err, "failed to fix permissions")
		}
	}

	defer func() {
		fmt.Fprintln(streams.Out, "Successfully enrolled the Elastic Agent.")
	}()

	if c.agentProc == nil {
		if c.daemonReload(ctx) != nil {
			c.log.Info("Elastic Agent might not be running; unable to trigger restart")
		} else {
			c.log.Info("Successfully triggered restart on running Elastic Agent.")
		}
		return nil
	}
	c.log.Info("Elastic Agent has been enrolled; start Elastic Agent")
	return nil
}

func (c *enrollCmd) writeDelayEnroll(streams *cli.IOStreams) error {
	enrollPath := paths.AgentEnrollFile()
	data, err := yaml.Marshal(c.options)
	if err != nil {
		return errors.New(
			err,
			"failed to marshall enrollment options",
			errors.TypeConfig,
			errors.M("path", enrollPath))
	}
	err = ioutil.WriteFile(enrollPath, data, 0600)
	if err != nil {
		return errors.New(
			err,
			"failed to write enrollment options file",
			errors.TypeFilesystem,
			errors.M("path", enrollPath))
	}
	fmt.Fprintf(streams.Out, "Successfully wrote %s for delayed enrollment of the Elastic Agent.\n", enrollPath)
	return nil
}

func (c *enrollCmd) fleetServerBootstrap(ctx context.Context, persistentConfig map[string]interface{}) (string, error) {
	c.log.Debug("verifying communication with running Elastic Agent daemon")
	agentRunning := true
	_, err := getDaemonStatus(ctx)
	if err != nil {
		if !c.options.FleetServer.SpawnAgent {
			// wait longer to try and communicate with the Elastic Agent
			err = waitForAgent(ctx, c.options.DaemonTimeout)
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

	agentConfig, err := c.createAgentConfig("", persistentConfig, c.options.FleetServer.Headers)
	if err != nil {
		return "", err
	}

	fleetConfig, err := createFleetServerBootstrapConfig(
		c.options.FleetServer.ConnStr, c.options.FleetServer.ServiceToken,
		c.options.FleetServer.PolicyID,
		c.options.FleetServer.Host, c.options.FleetServer.Port, c.options.FleetServer.InternalPort,
		c.options.FleetServer.Cert, c.options.FleetServer.CertKey, c.options.FleetServer.ElasticsearchCA, c.options.FleetServer.ElasticsearchCASHA256,
		c.options.FleetServer.Headers,
		c.options.ProxyURL,
		c.options.ProxyDisabled,
		c.options.ProxyHeaders,
		c.options.FleetServer.ElasticsearchInsecure,
	)
	if err != nil {
		return "", err
	}

	configToStore := map[string]interface{}{
		"agent": agentConfig,
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

	token, err := waitForFleetServer(ctx, agentSubproc, c.log, c.options.FleetServer.Timeout)
	if err != nil {
		return "", errors.New(err, "fleet-server failed", errors.TypeApplication)
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

	if c.options.FleetServer.InternalPort > 0 {
		if c.options.FleetServer.InternalPort != defaultFleetServerInternalPort {
			c.log.Warnf("Internal endpoint configured to: %d. Changing this value is not supported.", c.options.FleetServer.InternalPort)
		}
		c.options.InternalURL = fmt.Sprintf("%s:%d", defaultFleetServerInternalHost, c.options.FleetServer.InternalPort)
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
			c.options.FleetServer.Host, c.options.FleetServer.Port, c.options.FleetServer.InternalPort,
			c.options.FleetServer.Cert, c.options.FleetServer.CertKey, c.options.FleetServer.ElasticsearchCA, c.options.FleetServer.ElasticsearchCASHA256,
			c.options.FleetServer.Headers,
			c.options.ProxyURL, c.options.ProxyDisabled, c.options.ProxyHeaders,
			c.options.FleetServer.ElasticsearchInsecure,
		)
		if err != nil {
			return err
		}
		// no longer need bootstrap at this point
		serverConfig.Server.Bootstrap = false
		fleetConfig.Server = serverConfig.Server
		// use internal URL for future requests
		if c.options.InternalURL != "" {
			fleetConfig.Client.Host = c.options.InternalURL
		}
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
	if paths.Downloads() != "" {
		args = append(args, "--path.downloads", paths.Downloads())
	}
	if paths.Install() != "" {
		args = append(args, "--path.install", paths.Install())
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

func waitForAgent(ctx context.Context, timeout time.Duration) error {
	if timeout == 0 {
		timeout = 1 * time.Minute
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	maxBackoff := timeout
	if maxBackoff <= 0 {
		// indefinite timeout
		maxBackoff = 10 * time.Minute
	}

	resChan := make(chan waitResult)
	innerCtx, innerCancel := context.WithCancel(context.Background())
	defer innerCancel()
	go func() {
		backOff := expBackoffWithContext(innerCtx, 1*time.Second, maxBackoff)
		for {
			backOff.Wait()
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

func waitForFleetServer(ctx context.Context, agentSubproc <-chan *os.ProcessState, log *logger.Logger, timeout time.Duration) (string, error) {
	if timeout == 0 {
		timeout = 2 * time.Minute
	}
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	maxBackoff := timeout
	if maxBackoff <= 0 {
		// indefinite timeout
		maxBackoff = 10 * time.Minute
	}

	resChan := make(chan waitResult)
	innerCtx, innerCancel := context.WithCancel(context.Background())
	defer innerCancel()
	go func() {
		msg := ""
		msgCount := 0
		backExp := expBackoffWithContext(innerCtx, 1*time.Second, maxBackoff)
		for {
			backExp.Wait()
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
	port uint16, internalPort uint16,
	cert, key, esCA, esCASHA256 string,
	headers map[string]string,
	proxyURL string,
	proxyDisabled bool,
	proxyHeaders map[string]string,
	insecure bool,
) (*configuration.FleetAgentConfig, error) {
	localFleetServer := connStr != ""

	es, err := configuration.ElasticsearchFromConnStr(connStr, serviceToken, insecure)
	if err != nil {
		return nil, err
	}
	if esCA != "" {
		if es.TLS == nil {
			es.TLS = &tlscommon.Config{
				CAs: []string{esCA},
			}
		} else {
			es.TLS.CAs = []string{esCA}
		}
	}
	if esCASHA256 != "" {
		if es.TLS == nil {
			es.TLS = &tlscommon.Config{
				CATrustedFingerprint: esCASHA256,
			}
		} else {
			es.TLS.CATrustedFingerprint = esCASHA256
		}
	}
	if host == "" {
		host = defaultFleetServerHost
	}
	if port == 0 {
		port = defaultFleetServerPort
	}
	if internalPort == 0 {
		internalPort = defaultFleetServerInternalPort
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
	es.ProxyDisable = proxyDisabled
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
		if insecure {
			cfg.Server.TLS.VerificationMode = tlscommon.VerifyNone
		}
	}

	if localFleetServer {
		cfg.Client.Transport.Proxy.Disable = true
		cfg.Server.InternalPort = internalPort
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

func expBackoffWithContext(ctx context.Context, init, max time.Duration) backoff.Backoff {
	signal := make(chan struct{})
	bo := backoff.NewExpBackoff(signal, init, max)
	go func() {
		<-ctx.Done()
		close(signal)
	}()
	return bo
}
