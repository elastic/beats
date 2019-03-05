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

/*
Package filesystem provides a MetricSet implementation that fetches metrics
for each of the mounted file systems.

An example event looks as following:

{
    "@timestamp": "2016-05-23T08:05:34.853Z",
    "beat": {
        "hostname": "host.example.com",
        "name": "host.example.com"
    },
    "@metadata": {
        "beat": "noindex",
        "type": "doc"
    },
    "metricset": {
        "module": "system",
        "name": "filesystem",
        "rtt": 115
    },
    "system": {
        "filesystem": {
            "available": 105569656832,
            "device_name": "/dev/disk1",
            "type": "hfs",
            "files": 4294967279
            "free": 105831800832,
            "free_files": 4292793781,
            "mount_point": "/",
            "total": 249779191808,
            "used": {
                "bytes": 143947390976,
                "pct": 0.5763
            },
        }
    }
}

*/
package filesystem
