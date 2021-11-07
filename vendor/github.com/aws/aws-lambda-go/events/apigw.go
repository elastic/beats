// Copyright 2017 Amazon.com, Inc. or its affiliates. All Rights Reserved.

package events

// APIGatewayProxyRequest contains data coming from the API Gateway proxy
type APIGatewayProxyRequest struct {
	Resource                        string                        `json:"resource"` // The resource path defined in API Gateway
	Path                            string                        `json:"path"`     // The url path for the caller
	HTTPMethod                      string                        `json:"httpMethod"`
	Headers                         map[string]string             `json:"headers"`
	MultiValueHeaders               map[string][]string           `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string             `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string           `json:"multiValueQueryStringParameters"`
	PathParameters                  map[string]string             `json:"pathParameters"`
	StageVariables                  map[string]string             `json:"stageVariables"`
	RequestContext                  APIGatewayProxyRequestContext `json:"requestContext"`
	Body                            string                        `json:"body"`
	IsBase64Encoded                 bool                          `json:"isBase64Encoded,omitempty"`
}

// APIGatewayProxyResponse configures the response to be returned by API Gateway for the request
type APIGatewayProxyResponse struct {
	StatusCode        int                 `json:"statusCode"`
	Headers           map[string]string   `json:"headers"`
	MultiValueHeaders map[string][]string `json:"multiValueHeaders"`
	Body              string              `json:"body"`
	IsBase64Encoded   bool                `json:"isBase64Encoded,omitempty"`
}

// APIGatewayProxyRequestContext contains the information to identify the AWS account and resources invoking the
// Lambda function. It also includes Cognito identity information for the caller.
type APIGatewayProxyRequestContext struct {
	AccountID     string                    `json:"accountId"`
	ResourceID    string                    `json:"resourceId"`
	OperationName string                    `json:"operationName,omitempty"`
	Stage         string                    `json:"stage"`
	RequestID     string                    `json:"requestId"`
	Identity      APIGatewayRequestIdentity `json:"identity"`
	ResourcePath  string                    `json:"resourcePath"`
	Authorizer    map[string]interface{}    `json:"authorizer"`
	HTTPMethod    string                    `json:"httpMethod"`
	APIID         string                    `json:"apiId"` // The API Gateway rest API Id
}

// APIGatewayRequestIdentity contains identity information for the request caller.
type APIGatewayRequestIdentity struct {
	CognitoIdentityPoolID         string `json:"cognitoIdentityPoolId"`
	AccountID                     string `json:"accountId"`
	CognitoIdentityID             string `json:"cognitoIdentityId"`
	Caller                        string `json:"caller"`
	APIKey                        string `json:"apiKey"`
	APIKeyID                      string `json:"apiKeyId"`
	AccessKey                     string `json:"accessKey"`
	SourceIP                      string `json:"sourceIp"`
	CognitoAuthenticationType     string `json:"cognitoAuthenticationType"`
	CognitoAuthenticationProvider string `json:"cognitoAuthenticationProvider"`
	UserArn                       string `json:"userArn"`
	UserAgent                     string `json:"userAgent"`
	User                          string `json:"user"`
}

// APIGatewayWebsocketProxyRequest contains data coming from the API Gateway proxy
type APIGatewayWebsocketProxyRequest struct {
	Resource                        string                                 `json:"resource"` // The resource path defined in API Gateway
	Path                            string                                 `json:"path"`     // The url path for the caller
	HTTPMethod                      string                                 `json:"httpMethod"`
	Headers                         map[string]string                      `json:"headers"`
	MultiValueHeaders               map[string][]string                    `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string                      `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string                    `json:"multiValueQueryStringParameters"`
	PathParameters                  map[string]string                      `json:"pathParameters"`
	StageVariables                  map[string]string                      `json:"stageVariables"`
	RequestContext                  APIGatewayWebsocketProxyRequestContext `json:"requestContext"`
	Body                            string                                 `json:"body"`
	IsBase64Encoded                 bool                                   `json:"isBase64Encoded,omitempty"`
}

// APIGatewayWebsocketProxyRequestContext contains the information to identify
// the AWS account and resources invoking the Lambda function. It also includes
// Cognito identity information for the caller.
type APIGatewayWebsocketProxyRequestContext struct {
	AccountID          string                    `json:"accountId"`
	ResourceID         string                    `json:"resourceId"`
	Stage              string                    `json:"stage"`
	RequestID          string                    `json:"requestId"`
	Identity           APIGatewayRequestIdentity `json:"identity"`
	ResourcePath       string                    `json:"resourcePath"`
	Authorizer         interface{}               `json:"authorizer"`
	HTTPMethod         string                    `json:"httpMethod"`
	APIID              string                    `json:"apiId"` // The API Gateway rest API Id
	ConnectedAt        int64                     `json:"connectedAt"`
	ConnectionID       string                    `json:"connectionId"`
	DomainName         string                    `json:"domainName"`
	Error              string                    `json:"error"`
	EventType          string                    `json:"eventType"`
	ExtendedRequestID  string                    `json:"extendedRequestId"`
	IntegrationLatency string                    `json:"integrationLatency"`
	MessageDirection   string                    `json:"messageDirection"`
	MessageID          interface{}               `json:"messageId"`
	RequestTime        string                    `json:"requestTime"`
	RequestTimeEpoch   int64                     `json:"requestTimeEpoch"`
	RouteKey           string                    `json:"routeKey"`
	Status             string                    `json:"status"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestIdentity contains identity information for the request caller.
type APIGatewayCustomAuthorizerRequestTypeRequestIdentity struct {
	APIKey   string `json:"apiKey"`
	SourceIP string `json:"sourceIp"`
}

// APIGatewayCustomAuthorizerContext represents the expected format of an API Gateway custom authorizer response.
// Deprecated. Code should be updated to use the Authorizer map from APIGatewayRequestIdentity. Ex: Authorizer["principalId"]
type APIGatewayCustomAuthorizerContext struct {
	PrincipalID *string `json:"principalId"`
	StringKey   *string `json:"stringKey,omitempty"`
	NumKey      *int    `json:"numKey,omitempty"`
	BoolKey     *bool   `json:"boolKey,omitempty"`
}

// APIGatewayCustomAuthorizerRequestTypeRequestContext represents the expected format of an API Gateway custom authorizer response.
type APIGatewayCustomAuthorizerRequestTypeRequestContext struct {
	Path         string                                               `json:"path"`
	AccountID    string                                               `json:"accountId"`
	ResourceID   string                                               `json:"resourceId"`
	Stage        string                                               `json:"stage"`
	RequestID    string                                               `json:"requestId"`
	Identity     APIGatewayCustomAuthorizerRequestTypeRequestIdentity `json:"identity"`
	ResourcePath string                                               `json:"resourcePath"`
	HTTPMethod   string                                               `json:"httpMethod"`
	APIID        string                                               `json:"apiId"`
}

// APIGatewayCustomAuthorizerRequest contains data coming in to a custom API Gateway authorizer function.
type APIGatewayCustomAuthorizerRequest struct {
	Type               string `json:"type"`
	AuthorizationToken string `json:"authorizationToken"`
	MethodArn          string `json:"methodArn"`
}

// APIGatewayCustomAuthorizerRequestTypeRequest contains data coming in to a custom API Gateway authorizer function.
type APIGatewayCustomAuthorizerRequestTypeRequest struct {
	Type                            string                                              `json:"type"`
	MethodArn                       string                                              `json:"methodArn"`
	Resource                        string                                              `json:"resource"`
	Path                            string                                              `json:"path"`
	HTTPMethod                      string                                              `json:"httpMethod"`
	Headers                         map[string]string                                   `json:"headers"`
	MultiValueHeaders               map[string][]string                                 `json:"multiValueHeaders"`
	QueryStringParameters           map[string]string                                   `json:"queryStringParameters"`
	MultiValueQueryStringParameters map[string][]string                                 `json:"multiValueQueryStringParameters"`
	PathParameters                  map[string]string                                   `json:"pathParameters"`
	StageVariables                  map[string]string                                   `json:"stageVariables"`
	RequestContext                  APIGatewayCustomAuthorizerRequestTypeRequestContext `json:"requestContext"`
}

// APIGatewayCustomAuthorizerResponse represents the expected format of an API Gateway authorization response.
type APIGatewayCustomAuthorizerResponse struct {
	PrincipalID        string                           `json:"principalId"`
	PolicyDocument     APIGatewayCustomAuthorizerPolicy `json:"policyDocument"`
	Context            map[string]interface{}           `json:"context,omitempty"`
	UsageIdentifierKey string                           `json:"usageIdentifierKey,omitempty"`
}

// APIGatewayCustomAuthorizerPolicy represents an IAM policy
type APIGatewayCustomAuthorizerPolicy struct {
	Version   string
	Statement []IAMPolicyStatement
}

// IAMPolicyStatement represents one statement from IAM policy with action, effect and resource
type IAMPolicyStatement struct {
	Action   []string
	Effect   string
	Resource []string
}
