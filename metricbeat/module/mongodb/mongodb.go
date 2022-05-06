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

package mongodb

import (
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/beats/v7/metricbeat/mb/parse"
)

func init() {
	// Register the ModuleFactory function for the "mongodb" module.
	if err := mb.Registry.AddModule("mongodb", NewModule); err != nil {
		panic(err)
	}
}

// ModuleConfig contains the common configuration for this module
type ModuleConfig struct {
	Hosts []string          `config:"hosts"    validate:"nonzero,required"`
	TLS   *tlscommon.Config `config:"ssl"`

	Database string `config:"database"`

	Username string `config:"username"`
	Password string `config:"password"`

	Credentials struct {
		AuthMechanism           string            `config:"auth_mechanism"`
		AuthMechanismProperties map[string]string `config:"auth_mechanism_properties"`
		AuthSource              string            `config:"auth_source"`
		PasswordSet             bool              `config:"password_set"`
	} `config:"credentials"`
}

type Metricset struct {
	mb.BaseMetricSet
	Config ModuleConfig
}

type module struct {
	mb.BaseModule
}

// NewModule creates a new mb.Module instance and validates that at least one host has been
// specified
func NewModule(base mb.BaseModule) (mb.Module, error) {
	return &module{base}, nil
}

func NewMetricset(base mb.BaseMetricSet) (*Metricset, error) {
	// Validate that at least one host has been specified.
	config := ModuleConfig{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, fmt.Errorf("could not read config: %w", err)
	}

	return &Metricset{Config: config, BaseMetricSet: base}, nil
}

// ParseURL parses valid MongoDB URL strings into an mb.HostData instance
func ParseURL(module mb.Module, host string) (mb.HostData, error) {
	c := struct {
		Username    string `config:"username"`
		Password    string `config:"password"`
		Credentials struct {
			AuthMechanism           string            `config:"auth_mechanism"`
			AuthMechanismProperties map[string]string `config:"auth_mechanism_properties"`
			AuthSource              string            `config:"auth_source"`
			PasswordSet             bool              `config:"password_set"`
		} `config:"credentials"`
	}{}
	if err := module.UnpackConfig(&c); err != nil {
		return mb.HostData{}, err
	}

	if parts := strings.SplitN(host, "://", 2); len(parts) != 2 {
		// Add scheme.
		host = fmt.Sprintf("mongodb://%s", host)
	}

	// This doesn't use URLHostParserBuilder because MongoDB URLs can contain
	// multiple hosts separated by commas (mongodb://host1,host2,host3?options).
	u, err := url.Parse(host)
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing URL: %v", err)
	}

	parse.SetURLUser(u, c.Username, c.Password)

	clientOptions := options.Client()
	clientOptions.Auth = &options.Credential{
		AuthMechanism:           c.Credentials.AuthMechanism,
		AuthMechanismProperties: c.Credentials.AuthMechanismProperties,
		AuthSource:              c.Credentials.AuthSource,
		Username:                c.Username,
		Password:                c.Password,
		PasswordSet:             false,
	}
	clientOptions.ApplyURI(host)

	// https://docs.mongodb.com/manual/reference/connection-string/
	_, err = url.Parse(clientOptions.GetURI())
	if err != nil {
		return mb.HostData{}, fmt.Errorf("error parsing URL: %v", err)
	}

	return parse.NewHostDataFromURL(u), nil
}

// NewDirectSession estbalishes direct connections with a list of hosts. It uses the supplied
// dialInfo parameter as a template for establishing more direct connections
func NewDirectSession(uri string) (*mongo.Client, error) {
	clientOptions := options.Client().ApplyURI(uri)
	isDirectConnection := true
	clientOptions.Direct = &isDirectConnection

	return mongo.Connect(context.TODO(), clientOptions)
}

func NewClient(config ModuleConfig, timeout time.Duration) (*mongo.Client, error) {
	clientOptions := options.Client()
	clientOptions.Auth = &options.Credential{
		// TODO Support more auth mechanisms
		AuthMechanism: "",
		Username:      config.Username,
		Password:      config.Password,
		PasswordSet:   false,
	}
	directConnection := true
	clientOptions.Direct = &directConnection
	clientOptions.ConnectTimeout = &timeout

	return mongo.NewClient(clientOptions)
}
