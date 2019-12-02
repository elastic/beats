package azure

type azureInputConfig struct {
	// Kafka hosts with port, e.g. "localhost:9092"
	ConnectionString string `config:"connection_string" validate:"required"`
	EventHubName string `config:"eventhub" validate:"required"`
}

func defaultConfig() azureInputConfig {
	return azureInputConfig{
		ConnectionString: "",
		EventHubName: "",
	}
}
