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

package unix

import (
	"net"
	"time"

	input "github.com/elastic/beats/v8/filebeat/input/v2"
	stateless "github.com/elastic/beats/v8/filebeat/input/v2/input-stateless"
	"github.com/elastic/beats/v8/filebeat/inputsource"
	"github.com/elastic/beats/v8/filebeat/inputsource/unix"
	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/feature"
	"github.com/elastic/go-concert/ctxtool"
)

type server struct {
	unix.Server
	config
}

func Plugin() input.Plugin {
	return input.Plugin{
		Name:       "unix",
		Stability:  feature.Beta,
		Deprecated: false,
		Info:       "unix socket server",
		Manager:    stateless.NewInputManager(configure),
	}
}

func configure(cfg *common.Config) (stateless.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, err
	}

	return newServer(config)
}

func newServer(config config) (*server, error) {
	return &server{config: config}, nil
}

func (s *server) Name() string { return "unix" }

func (s *server) Test(_ input.TestContext) error {
	l, err := net.Listen("unix", s.config.Path)
	if err != nil {
		return err
	}
	return l.Close()
}

func (s *server) Run(ctx input.Context, publisher stateless.Publisher) error {
	log := ctx.Logger.Named("input.unix").With("path", s.config.Config.Path)

	log.Info("Starting Unix socket input")
	defer log.Info("Unix socket input stopped")

	cb := inputsource.NetworkFunc(func(data []byte, metadata inputsource.NetworkMetadata) {
		event := createEvent(data, metadata)
		publisher.Publish(event)
	})

	server, err := unix.New(log, &s.config.Config, cb)
	if err != nil {
		return err
	}

	log.Debugf("%s Input '%v' initialized", s.config.Config.SocketType, ctx.ID)

	err = server.Run(ctxtool.FromCanceller(ctx.Cancelation))

	// ignore error from 'Run' in case shutdown was signaled.
	if ctxerr := ctx.Cancelation.Err(); ctxerr != nil {
		err = ctxerr
	}
	return err
}

func createEvent(raw []byte, metadata inputsource.NetworkMetadata) beat.Event {
	return beat.Event{
		Timestamp: time.Now(),
		Fields: common.MapStr{
			"message": string(raw),
		},
	}
}
