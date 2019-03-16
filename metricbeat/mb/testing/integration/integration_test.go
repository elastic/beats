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

// +build integration

package integration

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/tests/compose"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/metricbeat/mb"
	mbtest "github.com/elastic/beats/metricbeat/mb/testing"
	// TODO: generate include file for these tests automatically moving forward
	_ "github.com/elastic/beats/metricbeat/module/php_fpm/pool"
	_ "github.com/elastic/beats/metricbeat/module/php_fpm/process"
)

type Config struct {
	Type        string
	URL         string
	Environment struct {
		Host struct {
			Env     string
			Default string
		}
		Port struct {
			Env     string
			Default string
		}
		Service string
	}
}

func TestAll(t *testing.T) {

	configFiles, _ := filepath.Glob(getModulesPath() + "/*/*/_meta/testdata/config.yml")

	for _, f := range configFiles {
		// get module and metricset name from path
		s := strings.Split(f, string(os.PathSeparator))
		moduleName := s[4]
		metricSetName := s[5]

		configFile, err := ioutil.ReadFile(f)
		if err != nil {
			log.Printf("yamlFile.Get err   #%v ", err)
		}
		var config Config
		err = yaml.Unmarshal(configFile, &config)
		if err != nil {
			log.Fatalf("Unmarshal: %v", err)
		}

		if config.Environment.Service == "" {
			continue
		}

		fmt.Println(moduleName)
		fmt.Println(metricSetName)
		Fetch(t, config, moduleName, metricSetName)
	}
}

func Fetch(t *testing.T, config Config, module, metricSet string) {
	compose.EnsureUp(t, config.Environment.Service)

	c := getConfig(module, metricSet, config.Environment.Host.Default+":"+config.Environment.Port.Default)

	var events []mb.Event
	var errs []error

	ms := mbtest.NewMetricSet(t, c)

	switch v := ms.(type) {
	case mb.ReportingMetricSetV2:
		f := mbtest.NewReportingMetricSetV2(t, c)
		events, errs = mbtest.ReportingFetchV2(f)
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			events[0].BeatEvent(module, metricSet).Fields.StringToPrint())
	case mb.ReportingMetricSetV2Error:
		f := mbtest.NewReportingMetricSetV2Error(t, c)
		events, errs = mbtest.ReportingFetchV2Error(f)
		t.Logf("%s/%s event: %+v", f.Module().Name(), f.Name(),
			events[0].BeatEvent(module, metricSet).Fields.StringToPrint())
	default:
		t.Fatalf("unknown type: %T", v)
	}

	assert.Empty(t, errs)
	if !assert.NotEmpty(t, events) {
		t.FailNow()
	}
}

func runTest(t *testing.T, file string, module, metricSetName, url string) {

}

// GetConfig returns config for elasticsearch module
func getConfig(module, metricSet, url string) map[string]interface{} {
	fmt.Println(url)
	return map[string]interface{}{
		"module":     module,
		"metricsets": []string{metricSet},
		"hosts":      []string{url},
	}
}

func getModulesPath() string {
	return "../../../module"
}

func getModulePath(module string) string {
	return getModulesPath() + "/" + module
}

func getMetricsetPath(module, metricSet string) string {
	return getModulePath(module) + "/" + metricSet
}
