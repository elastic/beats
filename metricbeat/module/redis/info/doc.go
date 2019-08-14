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
Package info fetches Redis server information and statistics using the Redis
INFO command.

The current implementation is tested with redis 3.2.3
More details on all the fields provided by the redis info command can be found here: http://redis.io/commands/INFO

`info.go` uses the Redis `INFO default` command for stats. This allows us to fetch  all metrics at once and filter out
undesired metrics based on user configuration on the client. The alternative would be to fetch each type as an
independent `INFO` call, which has the potential of introducing higher latency (e.g., more round trip Redis calls).

*/
package info
