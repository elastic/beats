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
Package process collects metrics about the running processes using information
from the operating system.

An example event looks as following:
{
  "@timestamp": "2016-05-25T20:57:51.854Z",
  "beat": {
    "hostname": "host.example.com",
    "name": "host.example.com"
  },
  "metricset": {
    "module": "system",
    "name": "process",
    "rtt": 12269
  },
  "system": {
    "process": {
      "cmdline": "/System/Library/CoreServices/ReportCrash",
      "cpu": {
        "start_time": "22:57",
        "total_p": 0
      },
      "mem": {
        "rss": 27123712,
        "rss_pct": 0.0016,
        "share": 0,
        "size": 2577522688
      },
      "name": "ReportCrash",
      "pid": 97801,
      "parent": {
        "pid": 1
      },
      "state": "running",
      "username": "elastic"
    }
  },
  "type": "metricsets"
}
*/
package process
