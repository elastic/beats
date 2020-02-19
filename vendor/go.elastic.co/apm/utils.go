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

package apm

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"

	"go.elastic.co/apm/internal/apmhostutil"
	"go.elastic.co/apm/internal/apmstrings"
	"go.elastic.co/apm/model"
)

var (
	currentProcess model.Process
	goAgent        = model.Agent{Name: "go", Version: AgentVersion}
	goLanguage     = model.Language{Name: "go", Version: runtime.Version()}
	goRuntime      = model.Runtime{Name: runtime.Compiler, Version: runtime.Version()}
	localSystem    model.System

	serviceNameInvalidRegexp = regexp.MustCompile("[^" + serviceNameValidClass + "]")
	labelKeyReplacer         = strings.NewReplacer(`.`, `_`, `*`, `_`, `"`, `_`)

	rtypeBool    = reflect.TypeOf(false)
	rtypeFloat64 = reflect.TypeOf(float64(0))
)

const (
	envHostname        = "ELASTIC_APM_HOSTNAME"
	envServiceNodeName = "ELASTIC_APM_SERVICE_NODE_NAME"

	serviceNameValidClass = "a-zA-Z0-9 _-"

	// At the time of writing, all keyword length limits
	// are 1024 runes, enforced by JSON Schema.
	stringLengthLimit = 1024

	// Non-keyword string fields are not limited in length
	// by JSON Schema, but we still truncate all strings.
	// Some strings, such as database statement, we explicitly
	// allow to be longer than others.
	longStringLengthLimit = 10000
)

func init() {
	currentProcess = getCurrentProcess()
	localSystem = getLocalSystem()
}

func getCurrentProcess() model.Process {
	ppid := os.Getppid()
	title, err := currentProcessTitle()
	if err != nil || title == "" {
		title = filepath.Base(os.Args[0])
	}
	return model.Process{
		Pid:   os.Getpid(),
		Ppid:  &ppid,
		Title: truncateString(title),
		Argv:  os.Args,
	}
}

func makeService(name, version, environment string) model.Service {
	service := model.Service{
		Name:        truncateString(name),
		Version:     truncateString(version),
		Environment: truncateString(environment),
		Agent:       &goAgent,
		Language:    &goLanguage,
		Runtime:     &goRuntime,
	}

	serviceNodeName := os.Getenv(envServiceNodeName)
	if serviceNodeName != "" {
		service.Node = &model.ServiceNode{ConfiguredName: truncateString(serviceNodeName)}
	}

	return service
}

func getLocalSystem() model.System {
	system := model.System{
		Architecture: runtime.GOARCH,
		Platform:     runtime.GOOS,
	}
	system.Hostname = os.Getenv(envHostname)
	if system.Hostname == "" {
		if hostname, err := os.Hostname(); err == nil {
			system.Hostname = hostname
		}
	}
	system.Hostname = truncateString(system.Hostname)
	if container, err := apmhostutil.Container(); err == nil {
		system.Container = container
	}
	system.Kubernetes = getKubernetesMetadata()
	return system
}

func getKubernetesMetadata() *model.Kubernetes {
	kubernetes, err := apmhostutil.Kubernetes()
	if err != nil {
		kubernetes = nil
	}
	namespace := os.Getenv("KUBERNETES_NAMESPACE")
	podName := os.Getenv("KUBERNETES_POD_NAME")
	podUID := os.Getenv("KUBERNETES_POD_UID")
	nodeName := os.Getenv("KUBERNETES_NODE_NAME")
	if namespace == "" && podName == "" && podUID == "" && nodeName == "" {
		return kubernetes
	}
	if kubernetes == nil {
		kubernetes = &model.Kubernetes{}
	}
	if namespace != "" {
		kubernetes.Namespace = namespace
	}
	if nodeName != "" {
		if kubernetes.Node == nil {
			kubernetes.Node = &model.KubernetesNode{}
		}
		kubernetes.Node.Name = nodeName
	}
	if podName != "" || podUID != "" {
		if kubernetes.Pod == nil {
			kubernetes.Pod = &model.KubernetesPod{}
		}
		if podName != "" {
			kubernetes.Pod.Name = podName
		}
		if podUID != "" {
			kubernetes.Pod.UID = podUID
		}
	}
	return kubernetes
}

func cleanLabelKey(k string) string {
	return labelKeyReplacer.Replace(k)
}

// makeLabelValue returns v as a value suitable for including
// in a label value. If v is numerical or boolean, then it will
// be returned as-is; otherwise the value will be returned as a
// string, using fmt.Sprint if necessary, and possibly truncated
// using truncateString.
func makeLabelValue(v interface{}) interface{} {
	switch v.(type) {
	case nil, bool, float32, float64,
		uint, uint8, uint16, uint32, uint64,
		int, int8, int16, int32, int64:
		return v
	case string:
		return truncateString(v.(string))
	}
	// Slow path. If v has a non-basic type whose underlying
	// type is convertible to bool or float64, return v as-is.
	// Otherwise, stringify.
	rtype := reflect.TypeOf(v)
	if rtype.ConvertibleTo(rtypeBool) || rtype.ConvertibleTo(rtypeFloat64) {
		// Custom type
		return v
	}
	return truncateString(fmt.Sprint(v))
}

func validateServiceName(name string) error {
	idx := serviceNameInvalidRegexp.FindStringIndex(name)
	if idx == nil {
		return nil
	}
	return errors.Errorf(
		"invalid service name %q: character %q is not in the allowed set (%s)",
		name, name[idx[0]], serviceNameValidClass,
	)
}

func sanitizeServiceName(name string) string {
	return serviceNameInvalidRegexp.ReplaceAllString(name, "_")
}

func truncateString(s string) string {
	s, _ = apmstrings.Truncate(s, stringLengthLimit)
	return s
}

func truncateLongString(s string) string {
	s, _ = apmstrings.Truncate(s, longStringLengthLimit)
	return s
}

func nextGracePeriod(p time.Duration) time.Duration {
	if p == -1 {
		return 0
	}
	for i := time.Duration(0); i < 6; i++ {
		if p == (i * i * time.Second) {
			return (i + 1) * (i + 1) * time.Second
		}
	}
	return p
}

// jitterDuration returns d +/- some multiple of d in the range [0,j].
func jitterDuration(d time.Duration, rng *rand.Rand, j float64) time.Duration {
	if d == 0 || j == 0 {
		return d
	}
	r := (rng.Float64() * j * 2) - j
	return d + time.Duration(float64(d)*r)
}

func durationMicros(d time.Duration) float64 {
	us := d / time.Microsecond
	ns := d % time.Microsecond
	return float64(us) + float64(ns)/1e9
}
