// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package tag

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	plc4go "github.com/apache/plc4x/plc4go/pkg/api"
	plc4xconfig "github.com/apache/plc4x/plc4go/pkg/api/config"
	"github.com/apache/plc4x/plc4go/pkg/api/drivers"
	"github.com/apache/plc4x/plc4go/pkg/api/model"
	"github.com/apache/plc4x/plc4go/pkg/api/values"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

func init() {
	mb.Registry.MustAddMetricSet("plc4x", "tag", New,
		mb.WithHostParser(hostParser),
		mb.DefaultMetricSet(),
	)
}

// hostParser keeps the PLC4X connection string unchanged, because its format
// depends on the driver, for example modbus-tcp://plc.example.com:502.
func hostParser(_ mb.Module, host string) (mb.HostData, error) {
	if _, err := driverOf(host); err != nil {
		return mb.HostData{}, err
	}
	_, address, _ := strings.Cut(host, "://")
	return mb.HostData{URI: host, SanitizedURI: host, Host: address}, nil
}

// MetricSet reads the configured tags from one PLC.
type MetricSet struct {
	mb.BaseMetricSet
	tags    []tagConfig
	timeout time.Duration

	mu         sync.Mutex
	manager    plc4go.PlcDriverManager
	connection plc4go.PlcConnection
}

// New creates the metricset. It validates the configuration but does not
// connect: the connection is opened on the first fetch.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The plc4x tag metricset is beta.")

	var cfg config
	if err := base.Module().UnpackConfig(&cfg); err != nil {
		return nil, err
	}
	timeout := base.Module().Config().Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	return &MetricSet{
		BaseMetricSet: base,
		tags:          cfg.Tags,
		timeout:       timeout,
	}, nil
}

// Fetch reads all configured tags in one request and reports one event per
// tag. A failed request closes the connection so the next fetch reconnects.
func (m *MetricSet) Fetch(ctx context.Context, r mb.ReporterV2) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.connection == nil || !m.connection.IsConnected() {
		m.closeConnection()
		if err := m.connect(ctx); err != nil {
			return fmt.Errorf("connecting to PLC %s: %w", m.connectionString(), err)
		}
	}

	// Each tag is read in its own request: PLC4X fails a whole request when one
	// of its tags fails, which would hide the values of all other tags.
	events := make([]mb.Event, 0, len(m.tags))
	var lastErr error
	failed := 0
	for i, t := range m.tags {
		code, value, err := m.readTag(ctx, tagName(i), t.Address)
		event := mb.Event{MetricSetFields: eventFields(t, code, value)}
		if err != nil {
			failed++
			lastErr = err
			event.Error = fmt.Errorf("reading tag %s: %w", t.Address, err)
		}
		events = append(events, event)
	}
	if failed == len(m.tags) {
		// No tag could be read, which points at the connection rather than at
		// the tags. Reconnect on the next fetch.
		m.closeConnection()
		return fmt.Errorf("reading tags from PLC %s: %w", m.connectionString(), lastErr)
	}

	for _, e := range events {
		if !r.Event(e) {
			return nil
		}
	}
	return nil
}

// readTag reads one tag. The response code is PlcResponseCode_OK when the
// value is valid.
func (m *MetricSet) readTag(ctx context.Context, name, address string) (model.PlcResponseCode, values.PlcValue, error) {
	req, err := m.connection.ReadRequestBuilder().AddTagAddress(name, address).Build()
	if err != nil {
		return model.PlcResponseCode_INVALID_ADDRESS, nil, err
	}

	readCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	select {
	case res := <-req.ExecuteWithContext(readCtx):
		if err := res.GetErr(); err != nil {
			return model.PlcResponseCode_INTERNAL_ERROR, nil, err
		}
		resp := res.GetResponse()
		return resp.GetResponseCode(name), resp.GetValue(name), nil
	case <-readCtx.Done():
		return model.PlcResponseCode_REQUEST_TIMEOUT, nil, readCtx.Err()
	}
}

// Close closes the PLC connection and the driver manager.
func (m *MetricSet) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closeConnection()
	if m.manager != nil {
		err := m.manager.Close()
		m.manager = nil
		return err
	}
	return nil
}

// globalLogOnce routes the global zerolog logger, which parts of PLC4X use
// instead of the logger passed to the driver manager, to the Beats logger.
var globalLogOnce sync.Once

func (m *MetricSet) connect(ctx context.Context) error {
	globalLogOnce.Do(func() {
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
		zlog.Logger = zerolog.New(logWriter{m})
	})
	if m.manager == nil {
		opts := []plc4xconfig.WithOption{plc4xconfig.WithCustomLogger(zerolog.New(logWriter{m}).Level(zerolog.WarnLevel))}
		m.manager = plc4go.NewPlcDriverManager(opts...)
		drivers.RegisterModbusTcpDriver(m.manager, opts...)
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()
	select {
	case res := <-m.manager.GetConnection(m.connectionString()):
		if err := res.GetErr(); err != nil {
			return err
		}
		m.connection = res.GetConnection()
	case <-ctx.Done():
		return ctx.Err()
	}
	m.Logger().Infof("Connected to PLC %s", m.connectionString())
	return nil
}

func (m *MetricSet) closeConnection() {
	if m.connection == nil {
		return
	}
	select {
	case <-m.connection.Close():
	case <-time.After(m.timeout):
		m.Logger().Debugf("Closing the connection to PLC %s timed out", m.connectionString())
	}
	m.connection = nil
}

// connectionString returns the configured PLC4X connection string.
func (m *MetricSet) connectionString() string {
	return m.HostData().URI
}

func tagName(i int) string {
	return "tag" + strconv.Itoa(i)
}

// logWriter forwards PLC4X log records to the metricset logger.
type logWriter struct {
	m *MetricSet
}

func (w logWriter) Write(p []byte) (int, error) {
	w.m.Logger().Warnf("plc4x: %s", p)
	return len(p), nil
}
