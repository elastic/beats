// Copyright 2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

// CognitoEvent contains data from an event sent from AWS Cognito Sync
type CognitoEvent struct {
	DatasetName    string                          `json:"datasetName"`
	DatasetRecords map[string]CognitoDatasetRecord `json:"datasetRecords"`
	EventType      string                          `json:"eventType"`
	IdentityID     string                          `json:"identityId"`
	IdentityPoolID string                          `json:"identityPoolId"`
	Region         string                          `json:"region"`
	Version        int                             `json:"version"`
}

// CognitoDatasetRecord represents a record from an AWS Cognito Sync event
type CognitoDatasetRecord struct {
	NewValue string `json:"newValue"`
	OldValue string `json:"oldValue"`
	Op       string `json:"op"`
}

// CognitoEventUserPoolsPreSignup is sent by AWS Cognito User Pools when a user attempts to register
// (sign up), allowing a Lambda to perform custom validation to accept or deny the registration request
type CognitoEventUserPoolsPreSignup struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPreSignupRequest  `json:"request"`
	Response CognitoEventUserPoolsPreSignupResponse `json:"response"`
}

// CognitoEventUserPoolsPostConfirmation is sent by AWS Cognito User Pools after a user is confirmed,
// allowing the Lambda to send custom messages or add custom logic.
type CognitoEventUserPoolsPostConfirmation struct {
	CognitoEventUserPoolsHeader
	Request  CognitoEventUserPoolsPostConfirmationRequest  `json:"request"`
	Response CognitoEventUserPoolsPostConfirmationResponse `json:"response"`
}

// CognitoEventUserPoolsCallerContext contains information about the caller
type CognitoEventUserPoolsCallerContext struct {
	AWSSDKVersion string `json:"awsSdkVersion"`
	ClientID      string `json:"clientId"`
}

// CognitoEventUserPoolsHeader contains common data from events sent by AWS Cognito User Pools
type CognitoEventUserPoolsHeader struct {
	Version       string                             `json:"version"`
	TriggerSource string                             `json:"triggerSource"`
	Region        string                             `json:"region"`
	UserPoolID    string                             `json:"userPoolId"`
	CallerContext CognitoEventUserPoolsCallerContext `json:"callerContext"`
	UserName      string                             `json:"userName"`
}

// CognitoEventUserPoolsPreSignupRequest contains the request portion of a PreSignup event
type CognitoEventUserPoolsPreSignupRequest struct {
	UserAttributes map[string]string `json:"userAttributes"`
	ValidationData map[string]string `json:"validationData"`
}

// CognitoEventUserPoolsPreSignupResponse contains the response portion of a PreSignup event
type CognitoEventUserPoolsPreSignupResponse struct {
	AutoConfirmUser bool `json:"autoConfirmUser"`
	AutoVerifyEmail bool `json:"autoVerifyEmail"`
	AutoVerifyPhone bool `json:"autoVerifyPhone"`
}

// CognitoEventUserPoolsPostConfirmationRequest contains the request portion of a PostConfirmation event
type CognitoEventUserPoolsPostConfirmationRequest struct {
	UserAttributes map[string]string `json:"userAttributes"`
}

// CognitoEventUserPoolsPostConfirmationResponse contains the response portion of a PostConfirmation event
type CognitoEventUserPoolsPostConfirmationResponse struct {
}
