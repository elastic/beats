package statemetrics

import "github.com/elastic/beats/metricbeat/helper/prometheus"

func getCertificateSigningRequestMapping() (mm map[string]prometheus.MetricMap, lm map[string]prometheus.LabelMap) {
	return map[string]prometheus.MetricMap{
			"kube_certificatesigningrequest_cert_length": prometheus.Metric("certificatesigningrequest.cert.length"),
			"kube_certificatesigningrequest_labels": prometheus.ExtendedInfoMetric(
				prometheus.Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "labels"},
			),
			"kube_certificatesigningrequest_condition": prometheus.Metric("certificatesigningrequest.condition.count"),
			"kube_certificatesigningrequest_created":   prometheus.Metric("certificatesigningrequest.created", prometheus.OpUnixTimestampValue()),
		},
		map[string]prometheus.LabelMap{
			"certificatesigningrequest": prometheus.KeyLabel("certificatesigningrequest.name"),
			"label_rotting":             prometheus.KeyLabel("certificatesigningrequest.label_rotting"),
			"condition":                 prometheus.KeyLabel("certificatesigningrequest.condition.status"),
		}
}
