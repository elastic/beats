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

// +build linux

package apmhostutil

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"go.elastic.co/apm/model"
)

const (
	systemdScopeSuffix = ".scope"
)

var (
	cgroupContainerInfoOnce  sync.Once
	cgroupContainerInfoError error
	kubernetes               *model.Kubernetes
	container                *model.Container

	kubepodsRegexp = regexp.MustCompile(
		"" +
			`(?:^/kubepods[\S]*/pod([^/]+)/$)|` +
			`(?:^/kubepods\.slice/kubepods-[^/]+\.slice/kubepods-[^/]+-pod([^/]+)\.slice/$)`,
	)

	containerIDRegexp = regexp.MustCompile(
		"^" +
			"[[:xdigit:]]{64}|" +
			"[[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4,}" +
			"$",
	)
)

func containerInfo() (*model.Container, error) {
	container, _, err := cgroupContainerInfo()
	return container, err
}

func kubernetesInfo() (*model.Kubernetes, error) {
	_, kubernetes, err := cgroupContainerInfo()
	if err == nil && kubernetes == nil {
		return nil, errors.New("could not determine kubernetes info")
	}
	return kubernetes, err
}

func cgroupContainerInfo() (*model.Container, *model.Kubernetes, error) {
	cgroupContainerInfoOnce.Do(func() {
		cgroupContainerInfoError = func() error {
			f, err := os.Open("/proc/self/cgroup")
			if err != nil {
				return err
			}
			defer f.Close()

			c, k, err := readCgroupContainerInfo(f)
			if err != nil {
				return err
			}
			if c == nil {
				return errors.New("could not determine container info")
			}
			container = c
			kubernetes = k
			return nil
		}()
	})
	return container, kubernetes, cgroupContainerInfoError
}

func readCgroupContainerInfo(r io.Reader) (*model.Container, *model.Kubernetes, error) {
	var container *model.Container
	var kubernetes *model.Kubernetes
	s := bufio.NewScanner(r)
	for s.Scan() {
		fields := strings.SplitN(s.Text(), ":", 3)
		if len(fields) != 3 {
			continue
		}
		cgroupPath := fields[2]

		// Depending on the filesystem driver used for cgroup
		// management, the paths in /proc/pid/cgroup will have
		// one of the following formats in a Docker container:
		//
		//   systemd: /system.slice/docker-<container-ID>.scope
		//   cgroupfs: /docker/<container-ID>
		//
		// In a Kubernetes pod, the cgroup path will look like:
		//
		//   systemd: /kubepods.slice/kubepods-<QoS-class>.slice/kubepods-<QoS-class>-pod<pod-UID>.slice/<container-iD>.scope
		//   cgroupfs: /kubepods/<QoS-class>/pod<pod-UID>/<container-iD>
		//
		dir, id := path.Split(cgroupPath)
		if strings.HasSuffix(id, systemdScopeSuffix) {
			id = id[:len(id)-len(systemdScopeSuffix)]
			if dash := strings.IndexRune(id, '-'); dash != -1 {
				id = id[dash+1:]
			}
		}
		if match := kubepodsRegexp.FindStringSubmatch(dir); match != nil {
			// By default, Kubernetes will set the hostname of
			// the pod containers to the pod name. Users that
			// override the name should use the Downard API to
			// override the pod name.
			hostname, _ := os.Hostname()
			uid := match[1]
			if uid == "" {
				// Systemd cgroup driver is being used,
				// so we need to unescape '_' back to '-'.
				uid = strings.Replace(match[2], "_", "-", -1)
			}
			kubernetes = &model.Kubernetes{
				Pod: &model.KubernetesPod{
					Name: hostname,
					UID:  uid,
				},
			}
			// We don't check the contents of the last path segment
			// when we've matched "^/kubepods"; we assume that it is
			// a valid container ID.
			container = &model.Container{ID: id}
		} else if containerIDRegexp.MatchString(id) {
			container = &model.Container{ID: id}
		}
	}
	if err := s.Err(); err != nil {
		return nil, nil, err
	}
	return container, kubernetes, nil
}
