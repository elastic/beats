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

// Package otelstorage implements [backend.Registry] backed by an OpenTelemetry storage
// extension. The concrete extension wiring (e.g. collector factory) lives with the caller;
// this package exposes registry and [storage.Client] adaptation only.
package otelstorage

import (
	"context"
	"fmt"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/extension"
	xstorage "go.opentelemetry.io/collector/extension/xextension/storage"
	"go.uber.org/zap"

	"github.com/elastic/beats/v7/libbeat/statestore/backend"
	"github.com/elastic/elastic-agent-libs/logp"
)

// Settings configures a Registry for a storage extension built from the given factory output.
type Settings struct {
	Config *filestorage.Config

	ReceiverID component.ID
	Logger     *logp.Logger
}

// DefaultFileStorageConfig returns the default configuration produced by the
// file_storage extension factory. Callers can adjust fields (e.g. Directory)
// before passing the config into Settings.
func DefaultFileStorageConfig() *filestorage.Config {
	cfg, ok := filestorage.NewFactory().CreateDefaultConfig().(*filestorage.Config)
	if !ok {
		panic("filestorage factory returned unexpected config type")
	}
	return cfg
}

type registry struct {
	ctx    context.Context
	ext    xstorage.Extension
	recvID component.ID
}

// NewFileStorage builds a [backend.Registry] using the configured file storage extension factory.
// Config must be populated by the caller.
func NewFileStorage(ctx context.Context, s Settings) (backend.Registry, error) {
	if s.Config == nil {
		return nil, fmt.Errorf("otelstorage: Config is required")
	}
	cfg := s.Config
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	ts := componenttest.NewNopTelemetrySettings()
	if s.Logger != nil {
		ts.Logger = zap.New(s.Logger.Core())
	}

	set := extension.Settings{
		ID:                component.MustNewID("file_storage"),
		TelemetrySettings: ts,
		BuildInfo:         component.NewDefaultBuildInfo(),
	}
	extAny, err := filestorage.NewFactory().Create(ctx, set, cfg)
	if err != nil {
		return nil, err
	}
	storExt, ok := extAny.(xstorage.Extension)
	if !ok {
		_ = extAny.Shutdown(ctx)
		return nil, fmt.Errorf("otelstorage: extension factory did not return a storage.Extension")
	}
	if err := storExt.Start(ctx, componenttest.NewNopHost()); err != nil {
		_ = storExt.Shutdown(ctx)
		return nil, err
	}
	return &registry{ctx: ctx, ext: storExt, recvID: s.ReceiverID}, nil
}

func (r *registry) Access(storeName string) (backend.Store, error) {
	cli, err := r.ext.GetClient(r.ctx, component.KindReceiver, r.recvID, storeName)
	if err != nil {
		return nil, err
	}
	return NewStoreFromClient(r.ctx, cli), nil
}

// NewRegistryFromExtension wraps a storage.Extension already started by the
// collector (e.g. the contrib file_storage extension). Each Access call obtains
// a storage.Client via GetClient and adapts it with NewStoreFromClient.
func NewRegistryFromExtension(ctx context.Context, ext xstorage.Extension, recvID component.ID) backend.Registry {
	return &registry{ctx: ctx, ext: ext, recvID: recvID}
}

func (r *registry) Close() error {
	return r.ext.Shutdown(r.ctx)
}
