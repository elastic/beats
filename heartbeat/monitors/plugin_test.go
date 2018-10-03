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

package monitors

import (
	"reflect"
	"testing"

	"github.com/elastic/beats/libbeat/common"
)

func Test_newPluginsReg(t *testing.T) {
	tests := []struct {
		name string
		want *pluginsReg
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newPluginsReg(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newPluginsReg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRegisterActive(t *testing.T) {
	type args struct {
		name    string
		builder PluginBuilder
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RegisterActive(tt.args.name, tt.args.builder)
		})
	}
}

func TestMonitorPluginAlreadyExistsError_Error(t *testing.T) {
	type fields struct {
		name    string
		typ     Type
		builder PluginBuilder
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := ErrPluginAlreadyExists{
				name:    tt.fields.name,
				typ:     tt.fields.typ,
				builder: tt.fields.builder,
			}
			if got := m.Error(); got != tt.want {
				t.Errorf("ErrPluginAlreadyExists.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pluginsReg_add(t *testing.T) {
	type fields struct {
		monitors map[string]pluginBuilder
	}
	type args struct {
		plugin pluginBuilder
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pluginsReg{
				monitors: tt.fields.monitors,
			}
			if err := r.add(tt.args.plugin); (err != nil) != tt.wantErr {
				t.Errorf("pluginsReg.add() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_pluginsReg_register(t *testing.T) {
	type fields struct {
		monitors map[string]pluginBuilder
	}
	type args struct {
		plugin pluginBuilder
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pluginsReg{
				monitors: tt.fields.monitors,
			}
			if err := r.register(tt.args.plugin); (err != nil) != tt.wantErr {
				t.Errorf("pluginsReg.register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_pluginsReg_get(t *testing.T) {
	type fields struct {
		monitors map[string]pluginBuilder
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   pluginBuilder
		want1  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pluginsReg{
				monitors: tt.fields.monitors,
			}
			got, got1 := r.get(tt.args.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pluginsReg.get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("pluginsReg.get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_pluginsReg_String(t *testing.T) {
	type fields struct {
		monitors map[string]pluginBuilder
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &pluginsReg{
				monitors: tt.fields.monitors,
			}
			if got := r.String(); got != tt.want {
				t.Errorf("pluginsReg.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_pluginBuilder_create(t *testing.T) {
	type fields struct {
		name    string
		typ     Type
		builder PluginBuilder
	}
	type args struct {
		cfg *common.Config
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []Job
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &pluginBuilder{
				name:    tt.fields.name,
				typ:     tt.fields.typ,
				builder: tt.fields.builder,
			}
			got, err := e.create(tt.args.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("pluginBuilder.create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("pluginBuilder.create() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		name string
		t    Type
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.t.String(); got != tt.want {
				t.Errorf("Type.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
