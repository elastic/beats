// Package management provides the integration of the collector with the
// elastic-agent via grpc.
package management

//go:generate godocdown -plain=false -output Readme.md

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	protobuf "github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/collector/internal/status"
	"github.com/elastic/elastic-agent-client/v7/pkg/client"
	"github.com/elastic/elastic-agent-client/v7/pkg/proto"
)

// ConfigManager establishes a connection with the elastic-agent.
type ConfigManager struct {
	log        *logp.Logger
	dialConfig agentDialConfig
	isManaged  bool

	mu        sync.Mutex
	rpcClient client.Client

	muStatus sync.Mutex
	status   status.State
}

// EventHandler is used to configure callbacks that can be triggered by RPC calls from the Agent to the Collector.
type EventHandler struct {
	// OnConfig is called when the Elastic Agent is requesting that the configuration
	// be set to the provided new value.
	//
	// XXX: Is this the complete configuration or a delta?
	OnConfig func(*common.Config) error

	OnStop func()
}

// Settings used to configure the ConfigManager.
type Settings struct {
	Enabled bool `config:"enabled" yaml:"enabled"`
}

type agentRPCListener struct {
	log            *logp.Logger
	handler        EventHandler
	statusReporter status.Reporter

	// ctx and cancel hold the reference to the cancellation context, allowing
	// the handler to signal the manager Run method to shut down on critical errors.
	// The context is owned by the manager.Run method.
	ctx    context.Context
	cancel context.CancelFunc

	// err stores the last critical error before shutdown. The manager returns this
	// error value on shutdown.
	// If the agent shuts down due to a critical error, it first must set err and
	// then call cancel to inform the manager.
	// The manager will only read err after the shutdown of the rpc client is completed.
	err error
}

type agentDialConfig struct {
	Addr        string
	Token       string
	ServerName  string
	Certificate tls.Certificate
	CAs         *x509.CertPool
}

func (settings Settings) IsManaged() bool {
	return settings.Enabled
}

func NewConfigManager(log *logp.Logger, settings Settings) (*ConfigManager, error) {
	var dialConfig agentDialConfig
	isManaged := settings.IsManaged()

	if isManaged {
		var err error
		dialConfig, err = readAgentDialConfig(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read the agent dial configuration: %w", err)
		}
	}

	log = log.Named("agent-client")
	return &ConfigManager{log: log, dialConfig: dialConfig, isManaged: isManaged}, nil
}

func readAgentDialConfig(r io.Reader) (agentDialConfig, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return agentDialConfig{}, err
	}

	var info proto.ConnInfo
	if err := protobuf.Unmarshal(data, &info); err != nil {
		return agentDialConfig{}, err
	}

	cert, err := tls.X509KeyPair(info.PeerCert, info.PeerKey)
	if err != nil {
		return agentDialConfig{}, err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(info.CaCert)

	return agentDialConfig{
		Addr:        info.Addr,
		Token:       info.Token,
		ServerName:  info.ServerName,
		Certificate: cert,
		CAs:         caCertPool,
	}, nil
}

func (m *ConfigManager) Run(ctx context.Context, handler EventHandler) error {
	if !m.isManaged {
		return nil
	}

	m.muStatus.Lock()
	status := m.status
	m.muStatus.Unlock()

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rpcClient != nil {
		panic("The manager is already active")
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	clientHandler := &agentRPCListener{
		log:            m.log,
		handler:        handler,
		statusReporter: m,
		ctx:            ctx,
		cancel:         cancel,
	}

	trans := credentials.NewTLS(&tls.Config{
		ServerName:   m.dialConfig.ServerName,
		Certificates: []tls.Certificate{m.dialConfig.Certificate},
		RootCAs:      m.dialConfig.CAs,
	})
	rpcClient := client.New(m.dialConfig.Addr, m.dialConfig.Token, clientHandler, nil, grpc.WithTransportCredentials(trans))

	err := rpcClient.Start(ctx)
	if err != nil {
		// If start fails to connect it seams to leak resources. Stop seems to be able to clean them up.
		// => always run Stop before handling/returning errors.
		rpcClient.Stop()
		return err
	}

	rpcClient.Status(encodeStatus(status))

	m.rpcClient = rpcClient
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		m.rpcClient = nil
	}()

	<-ctx.Done()
	rpcClient.Stop()
	return clientHandler.err
}

func (m *ConfigManager) UpdateStatus(status status.Status, msg string) {
	m.muStatus.Lock()
	defer m.muStatus.Unlock()
	if !m.status.Update(status, msg) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.rpcClient == nil {
		return
	}

	m.rpcClient.Status(encodeStatus(m.status))
	m.log.Info("Status change to %s: %s", m.status.Status, m.status.Message)
}

func (c *agentRPCListener) OnStop() {
	if fn := c.handler.OnStop; fn != nil {
		fn()
	}
}

func (c *agentRPCListener) OnError(err error) {
	// agent library uses contexts internally, but forwards errors without checking.
	// Let's try to filter out some internal signaling.
	if err != context.Canceled {
		c.log.Errorf("elastic-agent-client got error: %s", err)
	}

	// XXX: Error reporting is not fully clear. It seems like the agent library
	//      just continues working, without checking/knowing if the error is critical.
	//      At  times io.EOF seems to be eaten, but as it is not clear if/how the agent
	//      can reconnect, we at least try to shut down if we lost the connection.
	if err == io.EOF {
		c.OnStop()
	}
}

func (c *agentRPCListener) OnConfig(input string) {
	if c.handler.OnConfig == nil {
		return
	}

	c.statusReporter.UpdateStatus(status.Configuring, "Update configuration")

	cfg, err := parseAgentConfig(input)
	if err != nil {
		c.log.Error(err)
		c.statusReporter.UpdateStatus(status.Failed, err.Error())
		return
	}

	if err := c.handler.OnConfig(cfg); err != nil {
		c.statusReporter.UpdateStatus(status.Degraded, err.Error())
	}
}

func parseAgentConfig(input string) (*common.Config, error) {
	// XXX: do we need to do more here? Beats integration seems to support a blocklist
	return common.NewConfigFrom(input)
}

func encodeStatus(st status.State) (code proto.StateObserved_Status, msg string, extra map[string]interface{}) {
	switch st.Status {
	case status.Unknown:
		// unknown is reported as healthy, as the status is unknown
		code = proto.StateObserved_HEALTHY
	case status.Starting:
		code = proto.StateObserved_STARTING
	case status.Configuring:
		code = proto.StateObserved_CONFIGURING
	case status.Running:
		code = proto.StateObserved_HEALTHY
	case status.Degraded:
		code = proto.StateObserved_DEGRADED
	case status.Failed:
		code = proto.StateObserved_FAILED
	case status.Stopping:
		code = proto.StateObserved_STOPPING
	default:
		// unknown status, still reported as healthy
		code = proto.StateObserved_HEALTHY
	}

	return code, st.Message, nil
}

func defaultAgentConfigManagerSettings() Settings {
	return Settings{}
}
