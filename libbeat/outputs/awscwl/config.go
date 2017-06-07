package awscwl

import (
	//	"github.com/aws/aws-sdk-go/aws"
	//	"github.com/aws/aws-sdk-go/aws/credentials"
	//	"github.com/aws/aws-sdk-go/aws/session"
	//	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/elastic/beats/libbeat/outputs"
)

type awscwlConfig struct {
	Region              string              `config:"region"`
	LogGroupName        string              `config:"log_group_name"`
	LogStreamNamePrefix string              `config:"log_stream_name_prefix"`
	AccessKeyId         string              `config:"access_key_id"`
	SecretAccessKey     string              `config:"secret_access_key"`
	SessionToken        string              `config:"session_token"`
	Codec               outputs.CodecConfig `config:"codec"`
}

var (
	defaultConfig = awscwlConfig{
		AccessKeyId:     "",
		SecretAccessKey: "",
		SessionToken:    "",
	}
)

const (
	defaultBulkSize      = 1024
	defaultFlushInterval = 60
)
