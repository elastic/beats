// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package dgram

import (
	"context"
	"net"
	"time"

	"github.com/elastic/go-concert/ctxtool"
	"github.com/elastic/go-concert/unison"

	"github.com/elastic/beats/v7/filebeat/inputsource"
	"github.com/elastic/beats/v7/libbeat/common/cfgtype"
	"github.com/elastic/beats/v7/libbeat/logp"
)

const windowErrBuffer = "A message sent on a datagram socket was larger than the internal message" +
	" buffer or some other network limit, or the buffer used to receive a datagram into was smaller" +
	" than the datagram itself."

// ListenerFactory is used to craete connections based on the configuration.
type ListenerFactory func() (net.PacketConn, error)

type ListenerConfig struct {
	Timeout        time.Duration
	MaxMessageSize cfgtype.ByteSize
}

type Listener struct {
	log      *logp.Logger
	family   inputsource.Family
	config   *ListenerConfig
	listener ListenerFactory
	connect  HandlerFactory
	tg       unison.TaskGroup
}

func NewListener(
	f inputsource.Family,
	path string,
	connect HandlerFactory,
	listenerFactory ListenerFactory,
	config *ListenerConfig,
) *Listener {
	return &Listener{
		log:      logp.NewLogger(f.String()),
		family:   f,
		config:   config,
		listener: listenerFactory,
		connect:  connect,
		tg:       unison.TaskGroup{},
	}
}

func (l *Listener) Run(ctx context.Context) error {
	l.log.Info("Started listening for " + l.family.String() + " connection")

	for ctx.Err() == nil {
		l.doRun(ctx)
	}
	return nil
}

func (l *Listener) doRun(ctx context.Context) {
	conn, err := l.listener()
	if err != nil {
		l.log.Debugw("Cannot connect", "error", err)
		return
	}

	connCtx, connCancel := ctxtool.WithFunc(ctx, func() {
		conn.Close()
	})
	defer connCancel()

	err = l.connectAndRun(connCtx, conn)
	if err != nil {
		l.log.Debugw("Error while processing input", "error", err)
	}
}

func (l *Listener) Start() error {
	l.log.Info("Started listening for " + l.family.String() + " connection")

	conn, err := l.listener()
	if err != nil {
		return err
	}

	l.tg.Go(func(ctx context.Context) error {
		connCtx, connCancel := ctxtool.WithFunc(ctxtool.FromCanceller(ctx), func() {
			conn.Close()
		})
		defer connCancel()

		return l.connectAndRun(ctxtool.FromCanceller(connCtx), conn)
	})
	return nil
}

func (l *Listener) connectAndRun(ctx context.Context, conn net.PacketConn) error {
	defer l.log.Recover("Panic handling datagram")

	handler := l.connect(*l.config)
	for ctx.Err() == nil {
		err := handler(ctx, conn)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Listener) Stop() {
	l.log.Debug("Stopping datagram socket server for " + l.family.String())
	err := l.tg.Stop()
	if err != nil {
		l.log.Errorf("Error while stopping datagram socket server: %v", err)
	}
}
