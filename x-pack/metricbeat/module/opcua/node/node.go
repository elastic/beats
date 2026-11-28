// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gopcua/opcua"
	"github.com/gopcua/opcua/ua"

	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("opcua", "node", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// hostParser keeps the configured endpoint unchanged. OPC UA endpoints use the
// opc.tcp scheme, which the default URL host parser does not know.
func hostParser(_ mb.Module, host string) (mb.HostData, error) {
	if !strings.HasPrefix(host, "opc.tcp://") {
		return mb.HostData{}, fmt.Errorf("unsupported OPC UA endpoint %q: it must start with opc.tcp://", host)
	}
	return mb.HostData{URI: host, SanitizedURI: host, Host: strings.TrimPrefix(host, "opc.tcp://")}, nil
}

// MetricSet reads the value of the configured nodes from one OPC UA server.
type MetricSet struct {
	mb.BaseMetricSet
	config  config
	nodes   []node
	timeout time.Duration

	mu     sync.Mutex
	client *opcua.Client
}

// New creates the metricset. It validates the configuration but does not
// connect: the connection is opened on the first fetch.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfg := defaultConfig()
	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}

	nodes := make([]node, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		id, err := ua.ParseNodeID(n.ID)
		if err != nil {
			return nil, fmt.Errorf("invalid node id %q: %w", n.ID, err)
		}
		nodes = append(nodes, node{id: id, rawID: n.ID, label: n.Label})
	}

	timeout := base.Module().Config().Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &MetricSet{
		BaseMetricSet: base,
		config:        cfg,
		nodes:         nodes,
		timeout:       timeout,
	}, nil
}

// Fetch reads all configured nodes in one request and reports one event per
// node. A failed read closes the session so the next fetch reconnects.
func (m *MetricSet) Fetch(ctx context.Context, r mb.ReporterV2) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.client == nil {
		if err := m.connect(ctx); err != nil {
			return fmt.Errorf("connecting to OPC UA server %s: %w", m.endpoint(), err)
		}
	}

	req := &ua.ReadRequest{
		NodesToRead:        make([]*ua.ReadValueID, len(m.nodes)),
		TimestampsToReturn: ua.TimestampsToReturnBoth,
	}
	for i, n := range m.nodes {
		req.NodesToRead[i] = &ua.ReadValueID{NodeID: n.id, AttributeID: ua.AttributeIDValue}
	}

	readCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	resp, err := m.client.Read(readCtx, req)
	if err != nil {
		m.closeClient()
		return fmt.Errorf("reading nodes from OPC UA server %s: %w", m.endpoint(), err)
	}
	if len(resp.Results) != len(m.nodes) {
		m.closeClient()
		return fmt.Errorf("OPC UA server %s returned %d results for %d nodes", m.endpoint(), len(resp.Results), len(m.nodes))
	}

	for i, n := range m.nodes {
		if !r.Event(mb.Event{MetricSetFields: eventFields(n, resp.Results[i])}) {
			return nil
		}
	}
	return nil
}

// Close closes the OPC UA session.
func (m *MetricSet) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeClient()
	return nil
}

func (m *MetricSet) connect(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	opts := []opcua.Option{
		opcua.ApplicationName("Elastic Metricbeat"),
		opcua.ApplicationURI(m.config.ApplicationURI),
		opcua.SecurityPolicy(m.config.SecurityPolicy),
		opcua.SecurityModeString(m.config.SecurityMode),
		opcua.DialTimeout(m.timeout),
		opcua.RequestTimeout(m.timeout),
		opcua.AutoReconnect(false),
	}
	if m.config.ClientCertificate != "" {
		opts = append(opts,
			opcua.CertificateFile(m.config.ClientCertificate),
			opcua.PrivateKeyFile(m.config.ClientKey),
		)
	}

	// The server's endpoint description supplies its certificate and the
	// identity token policies. The session itself is opened on the configured
	// endpoint, because servers often advertise a host name that is only
	// resolvable inside their own network, such as a container host name.
	endpoints, err := opcua.GetEndpoints(ctx, m.endpoint())
	if err != nil {
		return fmt.Errorf("getting endpoints: %w", err)
	}
	ep, err := opcua.SelectEndpoint(endpoints, m.config.SecurityPolicy, ua.MessageSecurityModeFromString(m.config.SecurityMode))
	if err != nil {
		return fmt.Errorf("no endpoint matches security_policy %s and security_mode %s: %w", m.config.SecurityPolicy, m.config.SecurityMode, err)
	}
	if m.config.Username != "" {
		opts = append(opts,
			opcua.AuthUsername(m.config.Username, m.config.Password),
			opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeUserName),
		)
	} else {
		opts = append(opts,
			opcua.AuthAnonymous(),
			opcua.SecurityFromEndpoint(ep, ua.UserTokenTypeAnonymous),
		)
	}

	client, err := opcua.NewClient(m.endpoint(), opts...)
	if err != nil {
		return fmt.Errorf("creating client: %w", err)
	}
	if err := client.Connect(ctx); err != nil {
		return err
	}
	m.client = client
	m.Logger().Infof("Connected to OPC UA server %s with security policy %s and mode %s", m.endpoint(), m.config.SecurityPolicy, m.config.SecurityMode)

	m.readDisplayNames(ctx)
	return nil
}

// readDisplayNames reads the display name of every node once per connection.
// A failure is logged and does not prevent reading values.
func (m *MetricSet) readDisplayNames(ctx context.Context) {
	req := &ua.ReadRequest{NodesToRead: make([]*ua.ReadValueID, len(m.nodes))}
	for i, n := range m.nodes {
		req.NodesToRead[i] = &ua.ReadValueID{NodeID: n.id, AttributeID: ua.AttributeIDDisplayName}
	}
	resp, err := m.client.Read(ctx, req)
	if err != nil || len(resp.Results) != len(m.nodes) {
		m.Logger().Warnf("Reading display names from OPC UA server %s failed: %v", m.endpoint(), err)
		return
	}
	for i, res := range resp.Results {
		if res == nil || res.Status != ua.StatusOK || res.Value == nil {
			continue
		}
		if lt, ok := res.Value.Value().(*ua.LocalizedText); ok && lt != nil {
			m.nodes[i].name = lt.Text
		}
	}
}

// endpoint returns the configured OPC UA endpoint URL, including the opc.tcp
// scheme.
func (m *MetricSet) endpoint() string {
	return m.HostData().URI
}

func (m *MetricSet) closeClient() {
	if m.client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	defer cancel()
	if err := m.client.Close(ctx); err != nil && !errors.Is(err, context.Canceled) {
		m.Logger().Debugf("Closing OPC UA session with %s: %v", m.endpoint(), err)
	}
	m.client = nil
}
