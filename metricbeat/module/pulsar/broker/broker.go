package broker

import (
	p "github.com/elastic/beats/v7/metricbeat/helper/prometheus"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

//Broker MetricSet
type MetricSet struct {
	mb.BaseMetricSet
	prometheus p.Prometheus
	mapping    *p.MetricsMapping
}

//New returns a prometheus based metricset
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	prometheus, _ := p.NewPrometheusClient(base)

	return &MetricSet{
		BaseMetricSet: base,
		prometheus:    prometheus,
		mapping: &p.MetricsMapping{
			Metrics: map[string]p.MetricMap{
				//namespace
				"pulsar_entry_size_le_128":          p.Metric("namespace.entry.size.le.128"),
				"pulsar_entry_size_le_512":          p.Metric("namespace.entry.size.le.512"),
				"pulsar_entry_size_le_1_kb":         p.Metric("namespace.entry.size.le.1.kb"),
				"pulsar_entry_size_le_2_kb":         p.Metric("namespace.entry.size.le.2.kb"),
				"pulsar_entry_size_le_4_kb":         p.Metric("namespace.entry.size.le.4.kb"),
				"pulsar_entry_size_le_16_kb":        p.Metric("namespace.entry.size.le.16.kb"),
				"pulsar_entry_size_le_100_kb":       p.Metric("namespace.entry.size.le.100.kb"),
				"pulsar_entry_size_le_1_mb":         p.Metric("namespace.entry.size.le.1.mb"),
				"pulsar_entry_size_le_overflow":     p.Metric("namespace.entry.size.le.overflow"),
				"pulsar_producers_count":            p.Metric("namespace.producers.count"),
				"pulsar_rate_in":                    p.Metric("namespace.rate.in"),
				"pulsar_rate_out":                   p.Metric("namespace.rate.out"),
				"pulsar_replication_backlog":        p.Metric("namespace.replication.backlog"),
				"pulsar_replication_rate_in":        p.Metric("namespace.replication.rate.in"),
				"pulsar_replication_rate_out":       p.Metric("namespace.replication.rate.out"),
				"pulsar_replication_throughput_in":  p.Metric("namespace.replication.throughput.in"),
				"pulsar_replication_throughput_out": p.Metric("namespace.replication.throughput.out"),
				"pulsar_subscriptions_count":        p.Metric("namespace.subscriptions.count"),
				"pulsar_throughput_in":              p.Metric("namespace.throughput.in"),
				"pulsar_throughput_out":             p.Metric("namespace.throughput.out"),
				"pulsar_topics_count":               p.Metric("namespace.topics.count"),
				"pulsar_consumers_count":            p.Metric("namespace.consumers.count"),
				//topic
				"pulsar_in_bytes_total":                    p.Metric("topic.in.bytes.total"),
				"pulsar_in_messages_total":                 p.Metric("topic.in.messages.total"),
				"pulsar_out_messages_total":                p.Metric("topic.out.messages.total"),
				"pulsar_out_bytes_total":                   p.Metric("topic.out.bytes.total"),
				"pulsar_storage_backlog_quota_limit":       p.Metric("topic.storage.backlog.quota.limit"),
				"pulsar_storage_backlog_size":              p.Metric("topic.storage.backlog.size"),
				"pulsar_storage_read_rate":                 p.Metric("topic.storage.read.rate"),
				"pulsar_storage_offloaded_size":            p.Metric("topic.storage.offloaded.size"),
				"pulsar_storage_size":                      p.Metric("topic.storage.size"),
				"pulsar_storage_write_rate":                p.Metric("topic.storage.write.rate"),
				"pulsar_storage_write_latency_0_5":         p.Metric("topic.storage.write.latency.le.0.5"),
				"pulsar_storage_write_latency_le_1":        p.Metric("topic.storage.write.latency.le.1"),
				"pulsar_storage_write_latency_le_5":        p.Metric("topic.storage.write.latency.le.5"),
				"pulsar_storage_write_latency_le_10":       p.Metric("topic.storage.write.latency.le.10"),
				"pulsar_storage_write_latency_le_20":       p.Metric("topic.storage.write.latency.le.20"),
				"pulsar_storage_write_latency_le_50":       p.Metric("topic.storage.write.latency.le.50"),
				"pulsar_storage_write_latency_le_100":      p.Metric("topic.storage.write.latency.le.100"),
				"pulsar_storage_write_latency_le_200":      p.Metric("topic.storage.write.latency.le.200"),
				"pulsar_storage_write_latency_le_1000":     p.Metric("topic.storage.write.latency.le.1000"),
				"pulsar_storage_write_latency_le_overflow": p.Metric("topic.storage.write.latency.le.overflow"),

				//subscription
				"pulsar_subscription_back_log":                    p.Metric("subscription.back.log"),
				"pulsar_subscription_blocked_on_unacked_messages": p.Metric("subscription.blocked.on.unacked.messages"),
				"pulsar_subscription_delayed":                     p.Metric("subscription.delayed"),
				"pulsar_subscription_msg_rate_out":                p.Metric("subscription.msg.rate.out"),
				"pulsar_subscription_msg_rate_redeliver":          p.Metric("subscription.msg.rate.redeliver"),
				"pulsar_subscription_msg_throughput_out":          p.Metric("subscription.msg.throughput.out"),
				"pulsar_subscription_unacked_message":             p.Metric("subscription.unacked.message"),
			},
		},
	}, nil
}

// init registers the MetricSet with the central registry as soon as the program
// starts.
func init() {
	mb.Registry.MustAddMetricSet("pulsar", "broker", New,
		mb.WithHostParser(p.HostParser))
}

func (m *MetricSet) Fetch(r mb.ReporterV2) error {
	return m.prometheus.ReportProcessedMetrics(m.mapping, r)
}
