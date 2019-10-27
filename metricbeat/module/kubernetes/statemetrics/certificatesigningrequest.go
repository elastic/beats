package statemetrics

import "github.com/elastic/beats/metricbeat/helper/prometheus"

func getCertificateSigningRequestMapping() (mm map[string]prometheus.MetricMap, lm map[string]prometheus.LabelMap) {
	return map[string]prometheus.MetricMap{
			"kube_certificatesigningrequest_cert_length": prometheus.Metric("certificatesigningrequest.cert.length"),
			"kube_certificatesigningrequest_labels": prometheus.ExtendedMetric("certificatesigningrequest.labels",
				prometheus.Configuration{StoreNonMappedLabels: true, NonMappedLabelsPlacement: "labels"}),
			"kube_certificatesigningrequest_condition": prometheus.Metric("certificatesigningrequest.condition"),
			"kube_certificatesigningrequest_created":   prometheus.Metric("certificatesigningrequest.created", prometheus.OpUnixTimestampValue()),
		},
		map[string]prometheus.LabelMap{
			"certificatesigningrequest": prometheus.KeyLabel("name"),
			"label_rotting":             prometheus.KeyLabel("label_rotting"),
			"condition":                 prometheus.KeyLabel("condition"),
		}
}
