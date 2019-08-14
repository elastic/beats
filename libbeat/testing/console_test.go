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

package testing

import (
	"bufio"
	"bytes"
	"errors"
	"testing"

	"github.com/fatih/color"
	"github.com/stretchr/testify/assert"
)

func init() {
	color.NoColor = true
}

func TestConsoleDriverInfo(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	driver.Info("field", "value")

	output.Flush()
	assert.Equal(t, buffer.String(), "field: value\n")
}

func TestConsoleDriverWarn(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	driver.Warn("warning", "you got a warning")

	output.Flush()
	assert.Equal(t, buffer.String(), "warning... WARN you got a warning\n")
}

func TestConsoleDriverError(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	err := errors.New("This is an error")

	driver.Error("no error", nil)
	driver.Error("error", err)

	output.Flush()
	assert.Equal(t, buffer.String(), "no error... OK\nerror... ERROR This is an error\n")
}

func TestConsoleDriverFatal(t *testing.T) {
	var killed bool
	buffer, output, driver := createDriver(func() { killed = true })

	err := errors.New("This is an error")

	driver.Fatal("no error", nil)
	driver.Fatal("error", err)

	output.Flush()
	assert.True(t, killed)
	assert.Equal(t, buffer.String(), "no error... OK\nerror... ERROR This is an error\n")
}

func TestConsoleDriverRun(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	var called bool
	driver.Run("test", func(d Driver) {
		called = true
	})

	output.Flush()
	assert.True(t, called)
	assert.Equal(t, buffer.String(), "test...OK\n")
}

func TestConsoleDriverResult(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	driver.Run("test", func(d Driver) {
		d.Result("This is a multiline\nresult")
	})

	output.Flush()
	assert.Equal(t, buffer.String(), "test...OK\n  result: \n   This is a multiline\n   result\n\n")
}

func TestConsoleDriverRunWithReports(t *testing.T) {
	buffer, output, driver := createDriver(nil)

	var called bool
	err := errors.New("This is an error")
	driver.Run("test", func(d Driver) {
		called = true
		d.Info("field", "value")
		d.Error("error", err)
	})

	output.Flush()
	assert.True(t, called)
	assert.Equal(t, buffer.String(), "test...\n  field: value\n  error... ERROR This is an error\n")
}

func createDriver(killer func()) (*bytes.Buffer, *bufio.Writer, *ConsoleDriver) {
	buffer := bytes.NewBufferString("")
	output := bufio.NewWriter(buffer)
	var driver *ConsoleDriver
	if killer != nil {
		driver = NewConsoleDriverWithKiller(output, killer)
	} else {
		driver = NewConsoleDriver(output)
	}
	return buffer, output, driver
}
