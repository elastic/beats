package kafka

import (
	"github.com/Shopify/sarama"
	"github.com/elastic/beats/metricbeat/mb"
)

func GetClient(client sarama.Client, metricset mb.MetricSet) (sarama.Client, error) {

	if client == nil {
		config := sarama.NewConfig()
		config.Net.DialTimeout = metricset.Module().Config().Timeout
		config.Net.ReadTimeout = metricset.Module().Config().Timeout
		config.ClientID = "metricbeat"

		var err error
		client, err = sarama.NewClient([]string{metricset.Host()}, config)
		if err != nil {
			return nil, err
		}
	}
	return client, nil
}
