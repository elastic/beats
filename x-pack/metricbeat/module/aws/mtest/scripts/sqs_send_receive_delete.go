// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scripts

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"strconv"

	awssdk "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	sqstypes "github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

func getQueueUrls(svc *sqs.Client) ([]string, error) {
	// ListQueues
	listQueuesInput := &sqs.ListQueuesInput{}
	output, err := svc.ListQueues(context.TODO(), listQueuesInput)
	if err != nil {
		return nil, err
	}
	return output.QueueUrls, nil
}

func sendMessages(qURL string, svc *sqs.Client, idx int) error {
	sendMessageInput := &sqs.SendMessageInput{
		DelaySeconds: 10,
		MessageAttributes: map[string]sqstypes.MessageAttributeValue{
			"Title": {
				DataType:    awssdk.String("String"),
				StringValue: awssdk.String("The Whistler" + strconv.Itoa(idx)),
			},
			"Author": {
				DataType:    awssdk.String("String"),
				StringValue: awssdk.String("John Grisham" + strconv.Itoa(idx)),
			},
			"WeeksOn": {
				DataType:    awssdk.String("Number"),
				StringValue: awssdk.String("6" + strconv.Itoa(idx)),
			},
		},
		MessageBody: awssdk.String("Information about current NY Times fiction bestseller for week of 01/01/2019"),
		QueueUrl:    &qURL,
	}

	output, err := svc.SendMessage(context.TODO(), sendMessageInput)
	if err != nil {
		return err
	}

	fmt.Println("Succeed writing message ", *output.MessageId)
	return nil
}

func receiveMessages(qURL string, svc *sqs.Client) ([]sqstypes.Message, error) {
	receiveMessageInput := &sqs.ReceiveMessageInput{
		QueueUrl:            &qURL,
		MaxNumberOfMessages: 10,
		//VisibilityTimeout:   aws.Int64(20),  // 20 seconds
		//WaitTimeSeconds:     aws.Int64(0),
	}
	output, err := svc.ReceiveMessage(context.TODO(), receiveMessageInput)
	if err != nil {
		return nil, err
	}

	fmt.Println("Received # messages: " + strconv.Itoa(len(output.Messages)))
	return output.Messages, nil
}

func deleteMessage(qURL string, svc *sqs.Client, message sqstypes.Message) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      &qURL,
		ReceiptHandle: message.ReceiptHandle,
	}
	output, err := svc.DeleteMessage(context.TODO(), deleteMessageInput)
	if err != nil {
		return err
	}

	fmt.Println("DeleteMessage: ", output.ResultMetadata)
	return nil
}

func sqsSendReceiveDelete() {
	fmt.Println("Please setup AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and SESSION_TOKEN first. If a temp credentials are needed, please run getTempCreds.go first.")
	regionsList := []string{"us-west-1", "us-east-1"}
	accessKeyID := "FAKE-ACCESS-KEY-ID"
	secretAccessKey := "FAKE-SECRET-ACCESS-KEY"
	sessionToken := "FAKE-SESSION-TOKEN"

	awsConfig := awssdk.NewConfig()
	awsCreds := awssdk.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}

	awsConfig.Credentials = credentials.StaticCredentialsProvider{
		Value: awsCreds,
	}

	for _, regionName := range regionsList {
		awsConfig.Region = regionName
		svc := sqs.NewFromConfig(*awsConfig)
		queueURLs, err := getQueueUrls(svc)
		if err != nil {
			fmt.Println("Failed getQueueUrls: ", err)
		}

		for i, qURL := range queueURLs {
			//SEND
			errS := sendMessages(qURL, svc, i)
			if errS != nil {
				fmt.Println("Error sendMessageSQS", errS)
			}

			// RECEIVE
			messages, errR := receiveMessages(qURL, svc)
			if errR != nil {
				fmt.Println("Error receiveMessages", errR)
			}

			// DELETE
			if len(messages) > 0 {
				errD := deleteMessage(qURL, svc, messages[0])
				if errD != nil {
					fmt.Println("Error deleteMessage", errD)
				}
			}
		}
	}
}
