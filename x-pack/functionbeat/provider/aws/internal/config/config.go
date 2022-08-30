package config

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"os"
)

const fileName = "functionbeat.yml"

func errCheck(err error) {
	if err != nil {
		panic(err)
	}
}

func getAwsConfig() aws.Config {
	awsConfig, err := config.LoadDefaultConfig(context.TODO())
	errCheck(err)

	return awsConfig
}

func fileExists(fileName string) bool {
	info, err := os.Stat(fileName)
	if os.IsNotExist(err) {
		return false
	}

	return !info.IsDir()
}

func writeConfig(content []byte) {
	fmt.Println("Writing configuration")
	err := os.WriteFile(fileName, content, 0444)
	errCheck(err)
	fmt.Println("Done")
}

func getConfigFromASM(secretId string) {
	fmt.Println("Fetching configuration from SecretsManager")
	asmClient := secretsmanager.NewFromConfig(getAwsConfig())
	result, err := asmClient.GetSecretValue(context.TODO(), &secretsmanager.GetSecretValueInput{SecretId: &secretId})

	errCheck(err)
	writeConfig([]byte(*result.SecretString))
}

func getConfigFromS3(bucketName string, bucketKey string) {
	fmt.Println("Fetching configuration from S3")
	s3Client := s3.NewFromConfig(getAwsConfig())
	buffer := manager.NewWriteAtBuffer([]byte{})
	downloader := manager.NewDownloader(s3Client)
	_, err := downloader.Download(context.TODO(), buffer, &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(bucketKey),
	})

	errCheck(err)
	writeConfig(buffer.Bytes())
}

func Load() {
	if fileExists(fileName) {
		return
	}

	configSecretId := os.Getenv("FB_CONFIG_SECRET_ID")
	configS3BucketName := os.Getenv("FB_CONFIG_S3_BUCKET_NAME")
	configS3BucketKey := os.Getenv("FB_CONFIG_S3_BUCKET_KEY")

	if len(configSecretId) > 0 && len(configS3BucketName) > 0 {
		panic(fmt.Errorf("can only load config from S3 or SecretsManager. Not both"))
	}

	if len(configSecretId) > 0 {
		getConfigFromASM(configSecretId)
		return
	}

	if len(configS3BucketName) > 0 {
		if len(configS3BucketKey) == 0 {
			panic(fmt.Errorf("bucket Key must be provided"))
		}

		getConfigFromS3(configS3BucketName, configS3BucketKey)
		return
	}

	panic(fmt.Errorf("failed to find or load configuration"))
}
