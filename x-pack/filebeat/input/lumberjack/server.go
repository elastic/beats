// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lumberjack

import (
	"crypto/tls"
	"net"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/netutil"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
	"github.com/elastic/elastic-agent-libs/transport/tlscommon"
	"github.com/elastic/go-lumber/lj"
	lumber "github.com/elastic/go-lumber/server"
)

type server struct {
	config         config
	log            *logp.Logger
	publish        func(beat.Event)
	metrics        *inputMetrics
	ljSvr          lumber.Server
	ljSvrCloseOnce sync.Once
	bindAddress    string
}

func newServer(c config, log *logp.Logger, pub func(beat.Event), metrics *inputMetrics) (*server, error) {
	ljSvr, bindAddress, err := newLumberjack(c)
	if err != nil {
		return nil, err
	}

	if metrics == nil {
		metrics = newInputMetrics("", monitoring.NewRegistry())
	}

	bindURI := "tcp://" + bindAddress
	if c.TLS.IsEnabled() {
		bindURI = "tls://" + bindAddress
	}
	log.Infof(inputName+" is listening at %v.", bindURI)
	metrics.bindAddress.Set(bindURI)

	return &server{
		config:      c,
		log:         log,
		publish:     pub,
		metrics:     metrics,
		ljSvr:       ljSvr,
		bindAddress: bindAddress,
	}, nil
}

func (s *server) Close() error {
	var err error
	s.ljSvrCloseOnce.Do(func() {
		err = s.ljSvr.Close()
	})
	return err
}

func (s *server) Run() error {
	// Process batches until the input is stopped.
	for batch := range s.ljSvr.ReceiveChan() {
		s.processBatch(batch)
	}

	return nil
}

func (s *server) processBatch(batch *lj.Batch) {
	s.metrics.batchesReceivedTotal.Inc()

	if len(batch.Events) == 0 {
		batch.ACK()
		s.metrics.batchesACKedTotal.Inc()
		return
	}
	s.metrics.messagesReceivedTotal.Add(uint64(len(batch.Events)))

	// Track all the Beat events associated to the Lumberjack batch so that
	// the batch can be ACKed after the Beat events are delivered successfully.
	start := time.Now()
	acker := newBatchACKTracker(func() {
		batch.ACK()
		s.metrics.batchesACKedTotal.Inc()
		s.metrics.batchProcessingTime.Update(time.Since(start).Nanoseconds())
	})

	for _, ljEvent := range batch.Events {
		acker.Add()
		s.publish(makeEvent(batch.RemoteAddr, batch.TLS, ljEvent, acker))
	}

	// Mark the batch as "ready" after Beat events are generated for each
	// Lumberjack event.
	acker.Ready()
}

func makeEvent(remoteAddr string, tlsState *tls.ConnectionState, lumberjackEvent interface{}, acker *batchACKTracker) beat.Event {
	event := beat.Event{
		Timestamp: time.Now().UTC(),
		Fields: map[string]interface{}{
			"source": map[string]interface{}{
				"address": remoteAddr,
			},
			"lumberjack": lumberjackEvent,
		},
		Private: acker,
	}

	if tlsState != nil && len(tlsState.PeerCertificates) > 0 {
		event.Fields["tls"] = map[string]interface{}{
			"client": map[string]interface{}{
				"subject": tlsState.PeerCertificates[0].Subject.CommonName,
			},
		}
	}

	return event
}

func newLumberjack(c config) (lj lumber.Server, bindAddress string, err error) {
	// Setup optional TLS.
	var tlsConfig *tls.Config
	if c.TLS.IsEnabled() {
		elasticTLSConfig, err := tlscommon.LoadTLSServerConfig(c.TLS)
		if err != nil {
			return nil, "", err
		}

		// NOTE: Passing an empty string disables checking the client certificate for a
		// specific hostname.
		tlsConfig = elasticTLSConfig.BuildServerConfig("")
	}

	// Start listener.
	l, err := net.Listen("tcp", c.ListenAddress)
	if err != nil {
		return nil, "", err
	}
	if tlsConfig != nil {
		l = tls.NewListener(l, tlsConfig)
	}
	if c.MaxConnections > 0 {
		l = netutil.LimitListener(l, c.MaxConnections)
	}

	// Start lumberjack server.
	s, err := lumber.NewWithListener(l, makeLumberjackOptions(c)...)
	if err != nil {
		return nil, "", err
	}

	return s, l.Addr().String(), nil
}

func makeLumberjackOptions(c config) []lumber.Option {
	var opts []lumber.Option

	// Versions
	for _, p := range c.Versions {
		switch strings.ToLower(p) {
		case "v1":
			opts = append(opts, lumber.V1(true))
		case "v2":
			opts = append(opts, lumber.V2(true))
		}
	}

	if c.Keepalive > 0 {
		opts = append(opts, lumber.Keepalive(c.Keepalive))
	}

	if c.Timeout > 0 {
		opts = append(opts, lumber.Timeout(c.Keepalive))
	}

	return opts
}
