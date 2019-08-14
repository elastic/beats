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

package mgenv

import (
	"fmt"
	"os"
	"sort"
	"strconv"
)

// Var holds an environment variables name, default value and doc string.
type Var struct {
	name  string
	other string
	doc   string
}

var envVars = map[string]Var{}
var envKeys []string

func makeVar(name, other, doc string) Var {
	if v, exists := envVars[name]; exists {
		return v
	}

	v := Var{name, other, doc}
	envVars[name] = v
	envKeys = append(envKeys, name)
	sort.Strings(envKeys)
	return v
}

// MakeEnv builds a dictionary of defined environment variables, such that
// these can be passed to other processes (e.g. providers)
func MakeEnv() map[string]string {
	m := make(map[string]string, len(envVars))
	for k, v := range envVars {
		m[k] = v.Get()
	}
	return m
}

// Keys returns the keys of registered environment variables. The keys returned
// are sorted.
// Note: The returned slice must not be changed or appended to.
func Keys() []string {
	return envKeys
}

// Find returns a registered Var by name.
func Find(name string) (Var, bool) {
	v, ok := envVars[name]
	return v, ok
}

// String registers an environment variable and reads the current contents.
func String(name, other, doc string) string {
	v := makeVar(name, other, doc)
	return v.Get()
}

// Bool registers an environment variable and interprets the current variable as bool.
func Bool(name string, other bool, doc string) bool {
	v := makeVar(name, fmt.Sprint(other), doc)
	b, err := strconv.ParseBool(v.Get())
	return err == nil && b
}

// Name returns the environment variables name
func (v Var) Name() string { return v.name }

// Default returns the environment variables default value as string.
func (v Var) Default() string { return v.other }

// Doc returns the doc-string.
func (v Var) Doc() string { return v.doc }

// Get reads an environment variable. Get returns the default value if the
// variable is not present or empty.
func (v Var) Get() string {
	val := os.Getenv(v.name)
	if val == "" {
		return v.Default()
	}
	return val
}
