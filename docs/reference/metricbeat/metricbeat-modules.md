---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/metricbeat-modules.html
---

# Modules [metricbeat-modules]

This section contains detailed information about the metric collecting modules contained in Metricbeat. Each module contains one or multiple metricsets. More details about each module can be found under the links below.

| Modules | Dashboards | Metricsets |
| --- | --- | --- |
| [ActiveMQ](/reference/metricbeat/metricbeat-module-activemq.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [broker](/reference/metricbeat/metricbeat-metricset-activemq-broker.md) |
| [queue](/reference/metricbeat/metricbeat-metricset-activemq-queue.md) |
| [topic](/reference/metricbeat/metricbeat-metricset-activemq-topic.md) |
| [Aerospike](/reference/metricbeat/metricbeat-module-aerospike.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [namespace](/reference/metricbeat/metricbeat-metricset-aerospike-namespace.md) |
| [Airflow](/reference/metricbeat/metricbeat-module-airflow.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [statsd](/reference/metricbeat/metricbeat-metricset-airflow-statsd.md) [beta] |
| [Apache](/reference/metricbeat/metricbeat-module-apache.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [status](/reference/metricbeat/metricbeat-metricset-apache-status.md) |
| [AWS](/reference/metricbeat/metricbeat-module-aws.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [awshealth](/reference/metricbeat/metricbeat-metricset-aws-awshealth.md) [beta] |
| [billing](/reference/metricbeat/metricbeat-metricset-aws-billing.md) [beta] |
| [cloudwatch](/reference/metricbeat/metricbeat-metricset-aws-cloudwatch.md) |
| [dynamodb](/reference/metricbeat/metricbeat-metricset-aws-dynamodb.md) [beta] |
| [ebs](/reference/metricbeat/metricbeat-metricset-aws-ebs.md) |
| [ec2](/reference/metricbeat/metricbeat-metricset-aws-ec2.md) |
| [elb](/reference/metricbeat/metricbeat-metricset-aws-elb.md) |
| [kinesis](/reference/metricbeat/metricbeat-metricset-aws-kinesis.md) [beta] |
| [lambda](/reference/metricbeat/metricbeat-metricset-aws-lambda.md) |
| [natgateway](/reference/metricbeat/metricbeat-metricset-aws-natgateway.md) [beta] |
| [rds](/reference/metricbeat/metricbeat-metricset-aws-rds.md) |
| [s3_daily_storage](/reference/metricbeat/metricbeat-metricset-aws-s3_daily_storage.md) |
| [s3_request](/reference/metricbeat/metricbeat-metricset-aws-s3_request.md) |
| [sns](/reference/metricbeat/metricbeat-metricset-aws-sns.md) [beta] |
| [sqs](/reference/metricbeat/metricbeat-metricset-aws-sqs.md) |
| [transitgateway](/reference/metricbeat/metricbeat-metricset-aws-transitgateway.md) [beta] |
| [usage](/reference/metricbeat/metricbeat-metricset-aws-usage.md) [beta] |
| [vpn](/reference/metricbeat/metricbeat-metricset-aws-vpn.md) [beta] |
| [AWS Fargate](/reference/metricbeat/metricbeat-module-awsfargate.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [task_stats](/reference/metricbeat/metricbeat-metricset-awsfargate-task_stats.md) [beta] |
| [Azure](/reference/metricbeat/metricbeat-module-azure.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [app_insights](/reference/metricbeat/metricbeat-metricset-azure-app_insights.md) [beta] |
| [app_state](/reference/metricbeat/metricbeat-metricset-azure-app_state.md) [beta] |
| [billing](/reference/metricbeat/metricbeat-metricset-azure-billing.md) [beta] |
| [compute_vm](/reference/metricbeat/metricbeat-metricset-azure-compute_vm.md) |
| [compute_vm_scaleset](/reference/metricbeat/metricbeat-metricset-azure-compute_vm_scaleset.md) |
| [container_instance](/reference/metricbeat/metricbeat-metricset-azure-container_instance.md) |
| [container_registry](/reference/metricbeat/metricbeat-metricset-azure-container_registry.md) |
| [container_service](/reference/metricbeat/metricbeat-metricset-azure-container_service.md) |
| [database_account](/reference/metricbeat/metricbeat-metricset-azure-database_account.md) |
| [monitor](/reference/metricbeat/metricbeat-metricset-azure-monitor.md) |
| [storage](/reference/metricbeat/metricbeat-metricset-azure-storage.md) |
| [Beat](/reference/metricbeat/metricbeat-module-beat.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [state](/reference/metricbeat/metricbeat-metricset-beat-state.md) |
| [stats](/reference/metricbeat/metricbeat-metricset-beat-stats.md) |
| [Benchmark](/reference/metricbeat/metricbeat-module-benchmark.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [info](/reference/metricbeat/metricbeat-metricset-benchmark-info.md) [beta] |
| [Ceph](/reference/metricbeat/metricbeat-module-ceph.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [cluster_disk](/reference/metricbeat/metricbeat-metricset-ceph-cluster_disk.md) |
| [cluster_health](/reference/metricbeat/metricbeat-metricset-ceph-cluster_health.md) |
| [cluster_status](/reference/metricbeat/metricbeat-metricset-ceph-cluster_status.md) |
| [mgr_cluster_disk](/reference/metricbeat/metricbeat-metricset-ceph-mgr_cluster_disk.md) [beta] |
| [mgr_cluster_health](/reference/metricbeat/metricbeat-metricset-ceph-mgr_cluster_health.md) [beta] |
| [mgr_osd_perf](/reference/metricbeat/metricbeat-metricset-ceph-mgr_osd_perf.md) [beta] |
| [mgr_osd_pool_stats](/reference/metricbeat/metricbeat-metricset-ceph-mgr_osd_pool_stats.md) [beta] |
| [mgr_osd_tree](/reference/metricbeat/metricbeat-metricset-ceph-mgr_osd_tree.md) [beta] |
| [mgr_pool_disk](/reference/metricbeat/metricbeat-metricset-ceph-mgr_pool_disk.md) [beta] |
| [monitor_health](/reference/metricbeat/metricbeat-metricset-ceph-monitor_health.md) |
| [osd_df](/reference/metricbeat/metricbeat-metricset-ceph-osd_df.md) |
| [osd_tree](/reference/metricbeat/metricbeat-metricset-ceph-osd_tree.md) |
| [pool_disk](/reference/metricbeat/metricbeat-metricset-ceph-pool_disk.md) |
| [Cloudfoundry](/reference/metricbeat/metricbeat-module-cloudfoundry.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [container](/reference/metricbeat/metricbeat-metricset-cloudfoundry-container.md) [beta] |
| [counter](/reference/metricbeat/metricbeat-metricset-cloudfoundry-counter.md) [beta] |
| [value](/reference/metricbeat/metricbeat-metricset-cloudfoundry-value.md) [beta] |
| [CockroachDB](/reference/metricbeat/metricbeat-module-cockroachdb.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [status](/reference/metricbeat/metricbeat-metricset-cockroachdb-status.md) |
| [Consul](/reference/metricbeat/metricbeat-module-consul.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [agent](/reference/metricbeat/metricbeat-metricset-consul-agent.md) [beta] |
| [Containerd](/reference/metricbeat/metricbeat-module-containerd.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [blkio](/reference/metricbeat/metricbeat-metricset-containerd-blkio.md) [beta] |
| [cpu](/reference/metricbeat/metricbeat-metricset-containerd-cpu.md) [beta] |
| [memory](/reference/metricbeat/metricbeat-metricset-containerd-memory.md) [beta] |
| [Coredns](/reference/metricbeat/metricbeat-module-coredns.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [stats](/reference/metricbeat/metricbeat-metricset-coredns-stats.md) |
| [Couchbase](/reference/metricbeat/metricbeat-module-couchbase.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [bucket](/reference/metricbeat/metricbeat-metricset-couchbase-bucket.md) |
| [cluster](/reference/metricbeat/metricbeat-metricset-couchbase-cluster.md) |
| [node](/reference/metricbeat/metricbeat-metricset-couchbase-node.md) |
| [CouchDB](/reference/metricbeat/metricbeat-module-couchdb.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [server](/reference/metricbeat/metricbeat-metricset-couchdb-server.md) |
| [Docker](/reference/metricbeat/metricbeat-module-docker.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [container](/reference/metricbeat/metricbeat-metricset-docker-container.md) |
| [cpu](/reference/metricbeat/metricbeat-metricset-docker-cpu.md) |
| [diskio](/reference/metricbeat/metricbeat-metricset-docker-diskio.md) |
| [event](/reference/metricbeat/metricbeat-metricset-docker-event.md) |
| [healthcheck](/reference/metricbeat/metricbeat-metricset-docker-healthcheck.md) |
| [image](/reference/metricbeat/metricbeat-metricset-docker-image.md) |
| [info](/reference/metricbeat/metricbeat-metricset-docker-info.md) |
| [memory](/reference/metricbeat/metricbeat-metricset-docker-memory.md) |
| [network](/reference/metricbeat/metricbeat-metricset-docker-network.md) |
| [network_summary](/reference/metricbeat/metricbeat-metricset-docker-network_summary.md) [beta] |
| [Dropwizard](/reference/metricbeat/metricbeat-module-dropwizard.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [collector](/reference/metricbeat/metricbeat-metricset-dropwizard-collector.md) |
| [Elasticsearch](/reference/metricbeat/metricbeat-module-elasticsearch.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [ccr](/reference/metricbeat/metricbeat-metricset-elasticsearch-ccr.md) |
| [cluster_stats](/reference/metricbeat/metricbeat-metricset-elasticsearch-cluster_stats.md) |
| [enrich](/reference/metricbeat/metricbeat-metricset-elasticsearch-enrich.md) |
| [index](/reference/metricbeat/metricbeat-metricset-elasticsearch-index.md) |
| [index_recovery](/reference/metricbeat/metricbeat-metricset-elasticsearch-index_recovery.md) |
| [index_summary](/reference/metricbeat/metricbeat-metricset-elasticsearch-index_summary.md) |
| [ingest_pipeline](/reference/metricbeat/metricbeat-metricset-elasticsearch-ingest_pipeline.md) [beta] |
| [ml_job](/reference/metricbeat/metricbeat-metricset-elasticsearch-ml_job.md) |
| [node](/reference/metricbeat/metricbeat-metricset-elasticsearch-node.md) |
| [node_stats](/reference/metricbeat/metricbeat-metricset-elasticsearch-node_stats.md) |
| [pending_tasks](/reference/metricbeat/metricbeat-metricset-elasticsearch-pending_tasks.md) |
| [shard](/reference/metricbeat/metricbeat-metricset-elasticsearch-shard.md) |
| [Envoyproxy](/reference/metricbeat/metricbeat-module-envoyproxy.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [server](/reference/metricbeat/metricbeat-metricset-envoyproxy-server.md) |
| [Etcd](/reference/metricbeat/metricbeat-module-etcd.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [leader](/reference/metricbeat/metricbeat-metricset-etcd-leader.md) |
| [metrics](/reference/metricbeat/metricbeat-metricset-etcd-metrics.md) [beta] |
| [self](/reference/metricbeat/metricbeat-metricset-etcd-self.md) |
| [store](/reference/metricbeat/metricbeat-metricset-etcd-store.md) |
| [Google Cloud Platform](/reference/metricbeat/metricbeat-module-gcp.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [billing](/reference/metricbeat/metricbeat-metricset-gcp-billing.md) |
| [carbon](/reference/metricbeat/metricbeat-metricset-gcp-carbon.md) [beta] |
| [compute](/reference/metricbeat/metricbeat-metricset-gcp-compute.md) |
| [dataproc](/reference/metricbeat/metricbeat-metricset-gcp-dataproc.md) |
| [firestore](/reference/metricbeat/metricbeat-metricset-gcp-firestore.md) |
| [gke](/reference/metricbeat/metricbeat-metricset-gcp-gke.md) |
| [loadbalancing](/reference/metricbeat/metricbeat-metricset-gcp-loadbalancing.md) |
| [metrics](/reference/metricbeat/metricbeat-metricset-gcp-metrics.md) |
| [pubsub](/reference/metricbeat/metricbeat-metricset-gcp-pubsub.md) |
| [storage](/reference/metricbeat/metricbeat-metricset-gcp-storage.md) |
| [Golang](/reference/metricbeat/metricbeat-module-golang.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [expvar](/reference/metricbeat/metricbeat-metricset-golang-expvar.md) |
| [heap](/reference/metricbeat/metricbeat-metricset-golang-heap.md) |
| [Graphite](/reference/metricbeat/metricbeat-module-graphite.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [server](/reference/metricbeat/metricbeat-metricset-graphite-server.md) |
| [HAProxy](/reference/metricbeat/metricbeat-module-haproxy.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [info](/reference/metricbeat/metricbeat-metricset-haproxy-info.md) |
| [stat](/reference/metricbeat/metricbeat-metricset-haproxy-stat.md) |
| [HTTP](/reference/metricbeat/metricbeat-module-http.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [json](/reference/metricbeat/metricbeat-metricset-http-json.md) |
| [server](/reference/metricbeat/metricbeat-metricset-http-server.md) |
| [IBM MQ](/reference/metricbeat/metricbeat-module-ibmmq.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [qmgr](/reference/metricbeat/metricbeat-metricset-ibmmq-qmgr.md) [beta] |
| [IIS](/reference/metricbeat/metricbeat-module-iis.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [application_pool](/reference/metricbeat/metricbeat-metricset-iis-application_pool.md) |
| [webserver](/reference/metricbeat/metricbeat-metricset-iis-webserver.md) |
| [website](/reference/metricbeat/metricbeat-metricset-iis-website.md) |
| [Istio](/reference/metricbeat/metricbeat-module-istio.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [citadel](/reference/metricbeat/metricbeat-metricset-istio-citadel.md) [beta] |
| [galley](/reference/metricbeat/metricbeat-metricset-istio-galley.md) [beta] |
| [istiod](/reference/metricbeat/metricbeat-metricset-istio-istiod.md) [beta] |
| [mesh](/reference/metricbeat/metricbeat-metricset-istio-mesh.md) [beta] |
| [mixer](/reference/metricbeat/metricbeat-metricset-istio-mixer.md) [beta] |
| [pilot](/reference/metricbeat/metricbeat-metricset-istio-pilot.md) [beta] |
| [proxy](/reference/metricbeat/metricbeat-metricset-istio-proxy.md) [beta] |
| [Jolokia](/reference/metricbeat/metricbeat-module-jolokia.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [jmx](/reference/metricbeat/metricbeat-metricset-jolokia-jmx.md) |
| [Kafka](/reference/metricbeat/metricbeat-module-kafka.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [broker](/reference/metricbeat/metricbeat-metricset-kafka-broker.md) [beta] |
| [consumer](/reference/metricbeat/metricbeat-metricset-kafka-consumer.md) [beta] |
| [consumergroup](/reference/metricbeat/metricbeat-metricset-kafka-consumergroup.md) |
| [partition](/reference/metricbeat/metricbeat-metricset-kafka-partition.md) |
| [producer](/reference/metricbeat/metricbeat-metricset-kafka-producer.md) [beta] |
| [Kibana](/reference/metricbeat/metricbeat-module-kibana.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [cluster_actions](/reference/metricbeat/metricbeat-metricset-kibana-cluster_actions.md) [beta] |
| [cluster_rules](/reference/metricbeat/metricbeat-metricset-kibana-cluster_rules.md) [beta] |
| [node_actions](/reference/metricbeat/metricbeat-metricset-kibana-node_actions.md) [beta] |
| [node_rules](/reference/metricbeat/metricbeat-metricset-kibana-node_rules.md) [beta] |
| [stats](/reference/metricbeat/metricbeat-metricset-kibana-stats.md) |
| [status](/reference/metricbeat/metricbeat-metricset-kibana-status.md) |
| [Kubernetes](/reference/metricbeat/metricbeat-module-kubernetes.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [apiserver](/reference/metricbeat/metricbeat-metricset-kubernetes-apiserver.md) |
| [container](/reference/metricbeat/metricbeat-metricset-kubernetes-container.md) |
| [controllermanager](/reference/metricbeat/metricbeat-metricset-kubernetes-controllermanager.md) |
| [event](/reference/metricbeat/metricbeat-metricset-kubernetes-event.md) |
| [node](/reference/metricbeat/metricbeat-metricset-kubernetes-node.md) |
| [pod](/reference/metricbeat/metricbeat-metricset-kubernetes-pod.md) |
| [proxy](/reference/metricbeat/metricbeat-metricset-kubernetes-proxy.md) |
| [scheduler](/reference/metricbeat/metricbeat-metricset-kubernetes-scheduler.md) |
| [state_container](/reference/metricbeat/metricbeat-metricset-kubernetes-state_container.md) |
| [state_cronjob](/reference/metricbeat/metricbeat-metricset-kubernetes-state_cronjob.md) |
| [state_daemonset](/reference/metricbeat/metricbeat-metricset-kubernetes-state_daemonset.md) |
| [state_deployment](/reference/metricbeat/metricbeat-metricset-kubernetes-state_deployment.md) |
| [state_job](/reference/metricbeat/metricbeat-metricset-kubernetes-state_job.md) |
| [state_node](/reference/metricbeat/metricbeat-metricset-kubernetes-state_node.md) |
| [state_persistentvolumeclaim](/reference/metricbeat/metricbeat-metricset-kubernetes-state_persistentvolumeclaim.md) |
| [state_pod](/reference/metricbeat/metricbeat-metricset-kubernetes-state_pod.md) |
| [state_replicaset](/reference/metricbeat/metricbeat-metricset-kubernetes-state_replicaset.md) |
| [state_resourcequota](/reference/metricbeat/metricbeat-metricset-kubernetes-state_resourcequota.md) |
| [state_service](/reference/metricbeat/metricbeat-metricset-kubernetes-state_service.md) |
| [state_statefulset](/reference/metricbeat/metricbeat-metricset-kubernetes-state_statefulset.md) |
| [state_storageclass](/reference/metricbeat/metricbeat-metricset-kubernetes-state_storageclass.md) |
| [system](/reference/metricbeat/metricbeat-metricset-kubernetes-system.md) |
| [volume](/reference/metricbeat/metricbeat-metricset-kubernetes-volume.md) |
| [KVM](/reference/metricbeat/metricbeat-module-kvm.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [dommemstat](/reference/metricbeat/metricbeat-metricset-kvm-dommemstat.md) [beta] |
| [status](/reference/metricbeat/metricbeat-metricset-kvm-status.md) [beta] |
| [Linux](/reference/metricbeat/metricbeat-module-linux.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [conntrack](/reference/metricbeat/metricbeat-metricset-linux-conntrack.md) [beta] |
| [iostat](/reference/metricbeat/metricbeat-metricset-linux-iostat.md) [beta] |
| [ksm](/reference/metricbeat/metricbeat-metricset-linux-ksm.md) [beta] |
| [memory](/reference/metricbeat/metricbeat-metricset-linux-memory.md) [beta] |
| [pageinfo](/reference/metricbeat/metricbeat-metricset-linux-pageinfo.md) [beta] |
| [pressure](/reference/metricbeat/metricbeat-metricset-linux-pressure.md) [beta] |
| [rapl](/reference/metricbeat/metricbeat-metricset-linux-rapl.md) [beta] |
| [Logstash](/reference/metricbeat/metricbeat-module-logstash.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [node](/reference/metricbeat/metricbeat-metricset-logstash-node.md) |
| [node_stats](/reference/metricbeat/metricbeat-metricset-logstash-node_stats.md) |
| [Memcached](/reference/metricbeat/metricbeat-module-memcached.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [stats](/reference/metricbeat/metricbeat-metricset-memcached-stats.md) |
| [Cisco Meraki](/reference/metricbeat/metricbeat-module-meraki.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [device_health](/reference/metricbeat/metricbeat-metricset-meraki-device_health.md) [beta] |
| [MongoDB](/reference/metricbeat/metricbeat-module-mongodb.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [collstats](/reference/metricbeat/metricbeat-metricset-mongodb-collstats.md) |
| [dbstats](/reference/metricbeat/metricbeat-metricset-mongodb-dbstats.md) |
| [metrics](/reference/metricbeat/metricbeat-metricset-mongodb-metrics.md) |
| [replstatus](/reference/metricbeat/metricbeat-metricset-mongodb-replstatus.md) |
| [status](/reference/metricbeat/metricbeat-metricset-mongodb-status.md) |
| [MSSQL](/reference/metricbeat/metricbeat-module-mssql.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [performance](/reference/metricbeat/metricbeat-metricset-mssql-performance.md) |
| [transaction_log](/reference/metricbeat/metricbeat-metricset-mssql-transaction_log.md) |
| [Munin](/reference/metricbeat/metricbeat-module-munin.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [node](/reference/metricbeat/metricbeat-metricset-munin-node.md) |
| [MySQL](/reference/metricbeat/metricbeat-module-mysql.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [galera_status](/reference/metricbeat/metricbeat-metricset-mysql-galera_status.md) [beta] |
| [performance](/reference/metricbeat/metricbeat-metricset-mysql-performance.md) [beta] |
| [query](/reference/metricbeat/metricbeat-metricset-mysql-query.md) [beta] |
| [status](/reference/metricbeat/metricbeat-metricset-mysql-status.md) |
| [NATS](/reference/metricbeat/metricbeat-module-nats.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [connection](/reference/metricbeat/metricbeat-metricset-nats-connection.md) |
| [connections](/reference/metricbeat/metricbeat-metricset-nats-connections.md) |
| [route](/reference/metricbeat/metricbeat-metricset-nats-route.md) |
| [routes](/reference/metricbeat/metricbeat-metricset-nats-routes.md) |
| [stats](/reference/metricbeat/metricbeat-metricset-nats-stats.md) |
| [subscriptions](/reference/metricbeat/metricbeat-metricset-nats-subscriptions.md) |
| [Nginx](/reference/metricbeat/metricbeat-module-nginx.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [stubstatus](/reference/metricbeat/metricbeat-metricset-nginx-stubstatus.md) |
| [openai](/reference/metricbeat/metricbeat-module-openai.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [usage](/reference/metricbeat/metricbeat-metricset-openai-usage.md) [beta] |
| [Openmetrics](/reference/metricbeat/metricbeat-module-openmetrics.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [collector](/reference/metricbeat/metricbeat-metricset-openmetrics-collector.md) [beta] |
| [Oracle](/reference/metricbeat/metricbeat-module-oracle.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [performance](/reference/metricbeat/metricbeat-metricset-oracle-performance.md) |
| [sysmetric](/reference/metricbeat/metricbeat-metricset-oracle-sysmetric.md) [beta] |
| [tablespace](/reference/metricbeat/metricbeat-metricset-oracle-tablespace.md) |
| [Panw](/reference/metricbeat/metricbeat-module-panw.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [interfaces](/reference/metricbeat/metricbeat-metricset-panw-interfaces.md) [beta] |
| [routing](/reference/metricbeat/metricbeat-metricset-panw-routing.md) [beta] |
| [system](/reference/metricbeat/metricbeat-metricset-panw-system.md) [beta] |
| [vpn](/reference/metricbeat/metricbeat-metricset-panw-vpn.md) [beta] |
| [PHP_FPM](/reference/metricbeat/metricbeat-module-php_fpm.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [pool](/reference/metricbeat/metricbeat-metricset-php_fpm-pool.md) |
| [process](/reference/metricbeat/metricbeat-metricset-php_fpm-process.md) |
| [PostgreSQL](/reference/metricbeat/metricbeat-module-postgresql.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [activity](/reference/metricbeat/metricbeat-metricset-postgresql-activity.md) |
| [bgwriter](/reference/metricbeat/metricbeat-metricset-postgresql-bgwriter.md) |
| [database](/reference/metricbeat/metricbeat-metricset-postgresql-database.md) |
| [statement](/reference/metricbeat/metricbeat-metricset-postgresql-statement.md) |
| [Prometheus](/reference/metricbeat/metricbeat-module-prometheus.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [collector](/reference/metricbeat/metricbeat-metricset-prometheus-collector.md) |
| [query](/reference/metricbeat/metricbeat-metricset-prometheus-query.md) |
| [remote_write](/reference/metricbeat/metricbeat-metricset-prometheus-remote_write.md) |
| [RabbitMQ](/reference/metricbeat/metricbeat-module-rabbitmq.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [connection](/reference/metricbeat/metricbeat-metricset-rabbitmq-connection.md) |
| [exchange](/reference/metricbeat/metricbeat-metricset-rabbitmq-exchange.md) |
| [node](/reference/metricbeat/metricbeat-metricset-rabbitmq-node.md) |
| [queue](/reference/metricbeat/metricbeat-metricset-rabbitmq-queue.md) |
| [shovel](/reference/metricbeat/metricbeat-metricset-rabbitmq-shovel.md) [beta] |
| [Redis](/reference/metricbeat/metricbeat-module-redis.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [info](/reference/metricbeat/metricbeat-metricset-redis-info.md) |
| [key](/reference/metricbeat/metricbeat-metricset-redis-key.md) |
| [keyspace](/reference/metricbeat/metricbeat-metricset-redis-keyspace.md) |
| [Redis Enterprise](/reference/metricbeat/metricbeat-module-redisenterprise.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [node](/reference/metricbeat/metricbeat-metricset-redisenterprise-node.md) [beta] |
| [proxy](/reference/metricbeat/metricbeat-metricset-redisenterprise-proxy.md) [beta] |
| [SQL](/reference/metricbeat/metricbeat-module-sql.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [query](/reference/metricbeat/metricbeat-metricset-sql-query.md) |
| [Stan](/reference/metricbeat/metricbeat-module-stan.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [channels](/reference/metricbeat/metricbeat-metricset-stan-channels.md) |
| [stats](/reference/metricbeat/metricbeat-metricset-stan-stats.md) |
| [subscriptions](/reference/metricbeat/metricbeat-metricset-stan-subscriptions.md) |
| [Statsd](/reference/metricbeat/metricbeat-module-statsd.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [server](/reference/metricbeat/metricbeat-metricset-statsd-server.md) |
| [SyncGateway](/reference/metricbeat/metricbeat-module-syncgateway.md)  [beta] | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [db](/reference/metricbeat/metricbeat-metricset-syncgateway-db.md) [beta] |
| [memory](/reference/metricbeat/metricbeat-metricset-syncgateway-memory.md) [beta] |
| [replication](/reference/metricbeat/metricbeat-metricset-syncgateway-replication.md) [beta] |
| [resources](/reference/metricbeat/metricbeat-metricset-syncgateway-resources.md) [beta] |
| [System](/reference/metricbeat/metricbeat-module-system.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [core](/reference/metricbeat/metricbeat-metricset-system-core.md) |
| [cpu](/reference/metricbeat/metricbeat-metricset-system-cpu.md) |
| [diskio](/reference/metricbeat/metricbeat-metricset-system-diskio.md) |
| [entropy](/reference/metricbeat/metricbeat-metricset-system-entropy.md) |
| [filesystem](/reference/metricbeat/metricbeat-metricset-system-filesystem.md) |
| [fsstat](/reference/metricbeat/metricbeat-metricset-system-fsstat.md) |
| [load](/reference/metricbeat/metricbeat-metricset-system-load.md) |
| [memory](/reference/metricbeat/metricbeat-metricset-system-memory.md) |
| [network](/reference/metricbeat/metricbeat-metricset-system-network.md) |
| [network_summary](/reference/metricbeat/metricbeat-metricset-system-network_summary.md) [beta] |
| [process](/reference/metricbeat/metricbeat-metricset-system-process.md) |
| [process_summary](/reference/metricbeat/metricbeat-metricset-system-process_summary.md) |
| [raid](/reference/metricbeat/metricbeat-metricset-system-raid.md) |
| [service](/reference/metricbeat/metricbeat-metricset-system-service.md) [beta] |
| [socket](/reference/metricbeat/metricbeat-metricset-system-socket.md) |
| [socket_summary](/reference/metricbeat/metricbeat-metricset-system-socket_summary.md) |
| [uptime](/reference/metricbeat/metricbeat-metricset-system-uptime.md) |
| [users](/reference/metricbeat/metricbeat-metricset-system-users.md) [beta] |
| [Tomcat](/reference/metricbeat/metricbeat-module-tomcat.md)  [beta] | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [cache](/reference/metricbeat/metricbeat-metricset-tomcat-cache.md) [beta] |
| [memory](/reference/metricbeat/metricbeat-metricset-tomcat-memory.md) [beta] |
| [requests](/reference/metricbeat/metricbeat-metricset-tomcat-requests.md) [beta] |
| [threading](/reference/metricbeat/metricbeat-metricset-tomcat-threading.md) [beta] |
| [Traefik](/reference/metricbeat/metricbeat-module-traefik.md) | ![No prebuilt dashboards](images/icon-no.png "") |  |
|  |  | [health](/reference/metricbeat/metricbeat-metricset-traefik-health.md) |
| [uWSGI](/reference/metricbeat/metricbeat-module-uwsgi.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [status](/reference/metricbeat/metricbeat-metricset-uwsgi-status.md) |
| [vSphere](/reference/metricbeat/metricbeat-module-vsphere.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [cluster](/reference/metricbeat/metricbeat-metricset-vsphere-cluster.md) [beta] |
| [datastore](/reference/metricbeat/metricbeat-metricset-vsphere-datastore.md) |
| [datastorecluster](/reference/metricbeat/metricbeat-metricset-vsphere-datastorecluster.md) [beta] |
| [host](/reference/metricbeat/metricbeat-metricset-vsphere-host.md) |
| [network](/reference/metricbeat/metricbeat-metricset-vsphere-network.md) [beta] |
| [resourcepool](/reference/metricbeat/metricbeat-metricset-vsphere-resourcepool.md) [beta] |
| [virtualmachine](/reference/metricbeat/metricbeat-metricset-vsphere-virtualmachine.md) |
| [Windows](/reference/metricbeat/metricbeat-module-windows.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [perfmon](/reference/metricbeat/metricbeat-metricset-windows-perfmon.md) |
| [service](/reference/metricbeat/metricbeat-metricset-windows-service.md) |
| [wmi](/reference/metricbeat/metricbeat-metricset-windows-wmi.md) [beta] |
| [ZooKeeper](/reference/metricbeat/metricbeat-module-zookeeper.md) | ![Prebuilt dashboards are available](images/icon-yes.png "") |  |
|  |  | [connection](/reference/metricbeat/metricbeat-metricset-zookeeper-connection.md) |
| [mntr](/reference/metricbeat/metricbeat-metricset-zookeeper-mntr.md) |
| [server](/reference/metricbeat/metricbeat-metricset-zookeeper-server.md) |

