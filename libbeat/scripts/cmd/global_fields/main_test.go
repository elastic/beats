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

package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/libbeat/generator/fields"
)

type testcase struct {
	fieldsPath string
	files      []*fields.YmlFile
}

var (
	beatsPath     = filepath.Join("..", "..", "..", "..")
	filebeatFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "apache2", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "apache2", "access", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "apache2", "error", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "auditd", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "auditd", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "icinga", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "icinga", "debug", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "icinga", "main", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "icinga", "startup", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "iis", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "iis", "access", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "iis", "error", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "kafka", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "kafka", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "logstash", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "logstash", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "logstash", "slowlog", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "mongodb", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "mongodb", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "mysql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "mysql", "error", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "mysql", "slowlog", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "nginx", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "nginx", "access", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "nginx", "error", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "osquery", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "osquery", "result", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "postgresql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "postgresql", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "redis", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "redis", "log", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "redis", "slowlog", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "system", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "system", "auth", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "system", "syslog", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "traefik", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "filebeat", "module", "traefik", "access", "_meta", "fields.yml"),
			Indent: 8,
		},
	}
	heartbeatFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "heartbeat", "monitors", "active", "dialchain", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "heartbeat", "monitors", "active", "http", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "heartbeat", "monitors", "active", "icmp", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "heartbeat", "monitors", "active", "tcp", "_meta", "fields.yml"),
			Indent: 0,
		},
	}
	libbeatFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "libbeat", "processors", "add_cloud_metadata", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "libbeat", "processors", "add_docker_metadata", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "libbeat", "processors", "add_host_metadata", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "libbeat", "processors", "add_kubernetes_metadata", "_meta", "fields.yml"),
			Indent: 0,
		},
	}
	metricbeatFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "aerospike", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "aerospike", "namespace", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "apache", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "apache", "status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "cluster_disk", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "cluster_health", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "cluster_status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "monitor_health", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "osd_df", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "osd_tree", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "ceph", "pool_disk", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "couchbase", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "couchbase", "bucket", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "couchbase", "cluster", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "couchbase", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "container", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "cpu", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "diskio", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "healthcheck", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "image", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "info", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "memory", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "docker", "network", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "dropwizard", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "dropwizard", "collector", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "elasticsearch", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "elasticsearch", "index", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "elasticsearch", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "elasticsearch", "node_stats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "etcd", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "etcd", "leader", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "etcd", "self", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "etcd", "store", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "golang", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "golang", "expvar", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "golang", "heap", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "graphite", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "graphite", "server", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "haproxy", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "haproxy", "info", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "haproxy", "stat", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "http", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "http", "json", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "http", "server", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "jolokia", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "jolokia", "jmx", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kafka", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kafka", "consumergroup", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kafka", "partition", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kibana", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kibana", "status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "container", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "event", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "pod", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_container", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_deployment", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_pod", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_replicaset", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "state_statefulset", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "system", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kubernetes", "volume", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kvm", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "kvm", "dommemstat", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "logstash", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "logstash", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "logstash", "node_stats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "memcached", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "memcached", "stats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mongodb", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mongodb", "collstats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mongodb", "dbstats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mongodb", "status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "munin", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "munin", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mysql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "mysql", "status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "nginx", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "nginx", "stubstatus", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "php_fpm", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "php_fpm", "pool", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "postgresql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "postgresql", "activity", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "postgresql", "bgwriter", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "postgresql", "database", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "prometheus", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "prometheus", "collector", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "prometheus", "stats", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "rabbitmq", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "rabbitmq", "connection", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "rabbitmq", "node", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "rabbitmq", "queue", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "redis", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "redis", "info", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "redis", "keyspace", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "core", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "cpu", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "diskio", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "filesystem", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "fsstat", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "load", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "memory", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "network", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "process", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "process_summary", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "raid", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "socket", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "system", "uptime", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "uwsgi", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "uwsgi", "status", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "vsphere", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "vsphere", "datastore", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "vsphere", "host", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "vsphere", "virtualmachine", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "windows", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "windows", "perfmon", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "windows", "service", "_meta", "fields.yml"),
			Indent: 8,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "zookeeper", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "metricbeat", "module", "zookeeper", "mntr", "_meta", "fields.yml"),
			Indent: 8,
		},
	}
	packetbeatFiles = []*fields.YmlFile{
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "amqp", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "cassandra", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "dns", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "http", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "icmp", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "memcache", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "mongodb", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "mysql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "nfs", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "pgsql", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "redis", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "thrift", "_meta", "fields.yml"),
			Indent: 0,
		},
		&fields.YmlFile{
			Path:   filepath.Join(beatsPath, "packetbeat", "protos", "tls", "_meta", "fields.yml"),
			Indent: 0,
		},
	}
)

// TestCollectModuleFiles validates if the required files are collected
func TestCollectModuleFiles(t *testing.T) {
	cases := []testcase{
		testcase{
			fieldsPath: filepath.Join(beatsPath, "filebeat", "module"),
			files:      filebeatFiles,
		},
		testcase{
			fieldsPath: filepath.Join(beatsPath, "heartbeat", "monitors", "active"),
			files:      heartbeatFiles,
		},
		testcase{
			fieldsPath: filepath.Join(beatsPath, "libbeat", "processors"),
			files:      libbeatFiles,
		},
		testcase{
			fieldsPath: filepath.Join(beatsPath, "metricbeat", "module"),
			files:      metricbeatFiles,
		},
		testcase{
			fieldsPath: filepath.Join(beatsPath, "packetbeat", "protos"),
			files:      packetbeatFiles,
		},
	}

	for _, c := range cases {
		fieldFiles, err := fields.CollectModuleFiles(c.fieldsPath)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(t, reflect.DeepEqual(fieldFiles, c.files))
	}
}
