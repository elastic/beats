package azureeventhub

import "errors"

type azureInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	ConnectionString string `config:"connection_string" validate:"required"`
	EventHubName     string `config:"eventhub" validate:"required"`
	ConsumerGroup    string `config:"consumer_group"`
	EPHEnabled       bool   `config:"enable_eph"`
	SAName           string `config:"storage_account"`
	SAKey            string `config:"storage_account_key"`
	SAContainer      string `config:"storage_account_container"`
	// Azure Storage container to store leases and checkpoints

}

const ephContainerName = "ephcontainer"

// Validate validates the config.
func (conf *azureInputConfig) Validate() error {
	if conf.ConnectionString == "" {
		return errors.New("no connection string configured")
	}
	if conf.EventHubName == "" {
		return errors.New("no event hub name configured")
	}
	if conf.EPHEnabled {
		if conf.SAName == "" || conf.SAKey == "" {
			return errors.New("missing storage account information")
		}
		if conf.SAContainer == "" {
			conf.SAContainer = ephContainerName
		}
	}
	return nil
}
