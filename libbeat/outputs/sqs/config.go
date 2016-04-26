package sqs

type config struct {
	Index           string `config:"index"`
	SecretAccessKey string `config:"secret_access_key"`
	AccessKeyID     string `config:"access_key_id"`
	Region          string `config:"region"`
	QueueName       string `config:"queue_name"`
}

var (
	defaultConfig = config{}
)
