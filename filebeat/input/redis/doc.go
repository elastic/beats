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

// Package redis package contains input and harvester to read the redis slow log
//
// The redis slow log is stored in memory. The slow log can be activate on the redis command line as following:
//
// 	CONFIG SET slowlog-log-slower-than 2000000
//
// This sets the value of the slow log to 2000000 micro seconds (2s). All queries taking longer will be reported.
//
// As the slow log is in memory, it can be configured how many items it consists:
//
// 	CONFIG SET slowlog-max-len 200
//
// This sets the size of the slow log to 200 entries. In case the slow log is full, older entries are dropped.
package redis
