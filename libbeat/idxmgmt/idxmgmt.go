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

package idxmgmt

import (
	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/elasticsearch"
)

type SupportFactory func(beat.Info, *common.Config) (Supporter, error)

type Supporter interface {
	Enabled() bool
	ILM() ilm.Supporter
	Manager(client *elasticsearch.Client, fields []byte, migration bool) Manager
	BuildSelector(cfg *common.Config) (outputs.IndexSelector, error)
}

type Manager interface {
	Setup(template, policy bool) error
}

func DefaultSupport(info beat.Info, configRoot *common.Config) (Supporter, error) {
	factory := MakeDefaultSupport(nil)
	return factory(info, configRoot)
}

func MakeDefaultSupport(ilmSupport ilm.SupportFactory) SupportFactory {
	if ilmSupport == nil {
		ilmSupport = ilm.DefaultSupport
	}

	return func(info beat.Info, configRoot *common.Config) (Supporter, error) {
		cfg := struct {
			ILM      *common.Config `config:"setup.ilm"`
			Template *common.Config `common:"setup.template"`
		}{}
		if err := configRoot.Unpack(&cfg); err != nil {
			return nil, err
		}

		return newIndexSupport(info, ilmSupport, cfg.Template, cfg.ILM)
	}
}
