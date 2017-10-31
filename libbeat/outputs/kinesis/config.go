package kinesisout

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
)


type config struct {
	Region          string           `config:"region"`
	StreamName      string           `config:"stream"`
	PartitionKey    string           `config:"partition"`
	MaxRetries      int              `config:"retries"`
	Endpoint        string           `config:"endpoint"`
	LogLevel        aws.LogLevelType `config:"log"`
	DisableSSL      bool             `config:"nossl"`
	AccessKeyID     string           `config:"access_key_id"`
	SecretAccessKey string           `config:"secret_access_key"`
}

var defaultConfig = config{
	Region: "eu-west-1",
	PartitionKey: "static",
}

func (c *config) Validate() error {
	keyProvided := (c.AccessKeyID != "")
	secretProvided := (c.SecretAccessKey != "")
	if keyProvided != secretProvided {
		return errors.New("Must supply both a key and a secret, or neither to pass credentials via the environment")
	}
	return nil
}
