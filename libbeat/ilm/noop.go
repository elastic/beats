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

package ilm

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
)

type noopSupport struct{}
type noopManager struct{}

func NoopSupport(info beat.Info, config *common.Config) (Supporter, error) {
	return (*noopSupport)(nil), nil
}

func (*noopSupport) Mode() Mode                   { return ModeDisabled }
func (*noopSupport) Template() TemplateSettings   { return TemplateSettings{} }
func (*noopSupport) Manager(_ APIHandler) Manager { return (*noopManager)(nil) }

func (*noopManager) Enabled() (bool, error)    { return false, nil }
func (*noopManager) EnsureAlias() error        { return errOf(ErrOpNotAvailable) }
func (*noopManager) EnsurePolicy(_ bool) error { return errOf(ErrOpNotAvailable) }
