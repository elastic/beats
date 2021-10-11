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

//go:build integration
// +build integration

package filestream

const (
	elasticsearchMultilineLogs = `[2015-12-06 01:44:16,735][INFO ][node                     ] [Zach] version[2.0.0], pid[48553], build[de54438/2015-10-22T08:09:48Z]
[2015-12-06 01:44:16,736][INFO ][node                     ] [Zach] initializing ...
[2015-12-06 01:44:16,804][INFO ][plugins                  ] [Zach] loaded [], sites []
[2015-12-06 01:44:16,941][INFO ][env                      ] [Zach] using [1] data paths, mounts [[/ (/dev/disk1)]], net usable_space [66.3gb], net total_space [232.6gb], spins? [unknown], types [hfs]
[2015-12-06 01:44:19,177][INFO ][node                     ] [Zach] initialized
[2015-12-06 01:44:19,177][INFO ][node                     ] [Zach] starting ...
[2015-12-06 01:44:19,356][INFO ][transport                ] [Zach] publish_address {127.0.0.1:9300}, bound_addresses {127.0.0.1:9300}, {[fe80::1]:9300}, {[::1]:9300}
[2015-12-06 01:44:19,367][INFO ][discovery                ] [Zach] elasticsearch/qfPw9z0HQe6grbJQruTCJQ
[2015-12-06 01:44:22,405][INFO ][cluster.service          ] [Zach] new_master {Zach}{qfPw9z0HQe6grbJQruTCJQ}{127.0.0.1}{127.0.0.1:9300}, reason: zen-disco-join(elected_as_master, [0] joins received)
[2015-12-06 01:44:22,432][INFO ][http                     ] [Zach] publish_address {127.0.0.1:9200}, bound_addresses {127.0.0.1:9200}, {[fe80::1]:9200}, {[::1]:9200}
[2015-12-06 01:44:22,432][INFO ][node                     ] [Zach] started
[2015-12-06 01:44:22,446][INFO ][gateway                  ] [Zach] recovered [0] indices into cluster_state
[2015-12-06 01:44:52,882][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] creating index, cause [auto(bulk api)], templates [], shards [5]/[1], mappings [process, system]
[2015-12-06 01:44:53,256][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] update_mapping [process]
[2015-12-06 01:44:53,269][DEBUG][action.admin.indices.mapping.put] [Zach] failed to put mappings on indices [[filebeat-2015.12.06]], type [process]
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)
	at org.elasticsearch.cluster.service.InternalClusterService$UpdateTask.run(InternalClusterService.java:388)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedEsThreadPoolExecutor.java:225)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedEsThreadPoolExecutor.java:188)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
[2015-12-06 01:44:53,274][DEBUG][action.bulk              ] [Zach] [filebeat-2015.12.06][0] failed to execute bulk item (index) index {[filebeat-2015.12.06][process][AVF0v5vcVA0hoJdODMTz], source[{"@timestamp":"2015-12-06T00:44:52.448Z","beat":{"hostname":"ruflin","name":"ruflin"},"count":1,"proc":{"cpu":{"user":1902,"user_p":0,"system":941,"total":2843,"start_time":"Dec03"},"mem":{"size":3616309248,"rss":156405760,"rss_p":0.01,"share":0},"name":"Google Chrome H","pid":40572,"ppid":392,"state":"running"},"type":"process"}]}
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)
	at org.elasticsearch.cluster.service.InternalClusterService$UpdateTask.run(InternalClusterService.java:388)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedEsThreadPoolExecutor.java:225)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedEsThreadPoolExecutor.java:188)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
[2015-12-06 01:44:53,279][DEBUG][action.admin.indices.mapping.put] [Zach] failed to put mappings on indices [[filebeat-2015.12.06]], type [process]
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double], mapper [proc.cpu.user_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)
	at org.elasticsearch.cluster.service.InternalClusterService$UpdateTask.run(InternalClusterService.java:388)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedEsThreadPoolExecutor.java:225)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedEsThreadPoolExecutor.java:188)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
[2015-12-06 01:44:53,280][DEBUG][action.bulk              ] [Zach] [filebeat-2015.12.06][1] failed to execute bulk item (index) index {[filebeat-2015.12.06][process][AVF0v5vbVA0hoJdODMTj], source[{"@timestamp":"2015-12-06T00:44:52.416Z","beat":{"hostname":"ruflin","name":"ruflin"},"count":1,"proc":{"cpu":{"user":6643,"user_p":0.01,"system":693,"total":7336,"start_time":"01:44"},"mem":{"size":5182656512,"rss":248872960,"rss_p":0.01,"share":0},"name":"java","pid":48553,"ppid":48547,"state":"running"},"type":"process"}]}
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double], mapper [proc.cpu.user_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)
	at org.elasticsearch.cluster.service.InternalClusterService$UpdateTask.run(InternalClusterService.java:388)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedEsThreadPoolExecutor.java:225)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedEsThreadPoolExecutor.java:188)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
[2015-12-06 01:44:53,334][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] update_mapping [system]
[2015-12-06 01:44:53,646][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] create_mapping [filesystem]
`

	elasticsearchMultilineLongLogs = `[2015-12-06 01:44:16,735][INFO ][node                     ] [Zach] version[2.0.0], pid[48553], build[de54438/2015-10-22T08:09:48Z]
[2015-12-06 01:44:53,269][DEBUG][action.admin.indices.mapping.put] [Zach] failed to put mappings on indices [[filebeat-2015.12.06]], type [process]
MergeMappingException[Merge failed with failures {[mapper [proc.mem.rss_p] of different type, current_type [long], merged_type [double]]}]
	at org.elasticsearch.cluster.metadata.MetaDataMappingService$2.execute(MetaDataMappingService.java:388)
	at org.elasticsearch.cluster.service.InternalClusterService$UpdateTask.run(InternalClusterService.java:388)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.runAndClean(PrioritizedEsThreadPoolExecutor.java:225)
	at org.elasticsearch.common.util.concurrent.PrioritizedEsThreadPoolExecutor$TieBreakingPrioritizedRunnable.run(PrioritizedEsThreadPoolExecutor.java:188)
	at java.util.concurrent.ThreadPoolExecutor.runWorker(ThreadPoolExecutor.java:1142)
	at java.util.concurrent.ThreadPoolExecutor$Worker.run(ThreadPoolExecutor.java:617)
	at java.lang.Thread.run(Thread.java:745)
[2015-12-06 01:44:53,646][INFO ][cluster.metadata         ] [Zach] [filebeat-2015.12.06] create_mapping [filesystem]
`
)
