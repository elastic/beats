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

	secretConfigName := os.Getenv("FB_SECRET_CONFIG_NAME")
	s3ConfigBucketName := os.Getenv("FB_S3_CONFIG_BUCKET_NAME")
	s3ConfigBucketKey := os.Getenv("FB_S3_CONFIG_BUCKET_KEY")

	if len(secretConfigName) > 0 && len(s3ConfigBucketName) > 0 {
		panic(fmt.Errorf("can only load config from S3 or SecretsManager. Not both"))
	}

	if len(secretConfigName) > 0 {
		getConfigFromASM(secretConfigName)
		return
	}

	if len(s3ConfigBucketName) > 0 {
		if len(s3ConfigBucketKey) == 0 {
			panic(fmt.Errorf("bucket Key must be provided"))
		}

		getConfigFromS3(s3ConfigBucketName, s3ConfigBucketKey)
		return
	}

	panic(fmt.Errorf("failed to find or load configuration"))
}
