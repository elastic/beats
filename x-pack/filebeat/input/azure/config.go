package azure

import "errors"

type azureInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	ConnectionString string `config:"connection_string" validate:"required"`
	EventHubName string `config:"eventhub" validate:"required"`
	ConsumerGroup string `config:"consumer_group"`
}

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	if conf.ConnectionString =="" {
		return errors.New("no connection string configured")
	}
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	return nil
}
