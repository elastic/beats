// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package scripts

import (
	"context"
	"fmt"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/defaults"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

func getQueueUrls(svc sqsiface.ClientAPI) ([]string, error) {
	// ListQueues
	listQueuesInput := &sqs.ListQueuesInput{}
	req := svc.ListQueuesRequest(listQueuesInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		return nil, err
	}
	return output.QueueUrls, nil
}

func sendMessages(qURL string, svc sqsiface.ClientAPI, idx int) error {
	sendMessageInput := &sqs.SendMessageInput{
		DelaySeconds: aws.Int64(10),
		MessageAttributes: map[string]sqs.MessageAttributeValue{
			"Title": {
				DataType:    aws.String("String"),
				StringValue: aws.String("The Whistler" + strconv.Itoa(idx)),
			},
			"Author": {
				DataType:    aws.String("String"),
				StringValue: aws.String("John Grisham" + strconv.Itoa(idx)),
			},
			"WeeksOn": {
				DataType:    aws.String("Number"),
				StringValue: aws.String("6" + strconv.Itoa(idx)),
			},
		},
		MessageBody: aws.String("Information about current NY Times fiction bestseller for week of 01/01/2019"),
		QueueUrl:    &qURL,
	}

	req := svc.SendMessageRequest(sendMessageInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		return err
	}

	fmt.Println("Succeed writing message ", *output.MessageId)
	return nil
}

func receiveMessages(qURL string, svc sqsiface.ClientAPI) ([]sqs.Message, error) {
	receiveMessageInput := &sqs.ReceiveMessageInput{
		QueueUrl:            &qURL,
		MaxNumberOfMessages: aws.Int64(10),
		//VisibilityTimeout:   aws.Int64(20),  // 20 seconds
		//WaitTimeSeconds:     aws.Int64(0),
	}
	req := svc.ReceiveMessageRequest(receiveMessageInput)
	output, err := req.Send(context.TODO())
	if err != nil {
		return nil, err
	}

	fmt.Println("Received # messages: " + strconv.Itoa(len(output.Messages)))
	return output.Messages, nil
}

func deleteMessage(qURL string, svc sqsiface.ClientAPI, message sqs.Message) error {
	deleteMessageInput := &sqs.DeleteMessageInput{
		QueueUrl:      &qURL,
		ReceiptHandle: message.ReceiptHandle,
	}
	reqD := svc.DeleteMessageRequest(deleteMessageInput)
	output, err := reqD.Send(context.TODO())
	if err != nil {
		return err
	}

	fmt.Println("DeleteMessage: ", output.SDKResponseMetdata().Request.RequestID)
	return nil
}

func sqsSendReceiveDelete() {
	fmt.Println("Please setup AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY and SESSION_TOKEN first. If a temp credentials are needed, please run getTempCreds.go first.")
	regionsList := []string{"us-west-1", "us-east-1"}
	accessKeyID := "FAKE-ACCESS-KEY-ID"
	secretAccessKey := "FAKE-SECRET-ACCESS-KEY"
	sessionToken := "FAKE-SESSION-TOKEN"

	awsConfig := defaults.Config()
	awsCreds := aws.Credentials{
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		SessionToken:    sessionToken,
	}

	awsConfig.Credentials = aws.StaticCredentialsProvider{
		Value: awsCreds,
	}

	for _, regionName := range regionsList {
		awsConfig.Region = regionName
		svc := sqs.New(awsConfig)
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
