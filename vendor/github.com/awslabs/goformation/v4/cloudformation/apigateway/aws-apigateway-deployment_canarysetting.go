package apigateway

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Deployment_CanarySetting AWS CloudFormation Resource (AWS::ApiGateway::Deployment.CanarySetting)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-canarysetting.html
type Deployment_CanarySetting struct {

	// PercentTraffic AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-canarysetting.html#cfn-apigateway-deployment-canarysetting-percenttraffic
	PercentTraffic float64 `json:"PercentTraffic,omitempty"`

	// StageVariableOverrides AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-canarysetting.html#cfn-apigateway-deployment-canarysetting-stagevariableoverrides
	StageVariableOverrides map[string]string `json:"StageVariableOverrides,omitempty"`

	// UseStageCache AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-canarysetting.html#cfn-apigateway-deployment-canarysetting-usestagecache
	UseStageCache bool `json:"UseStageCache,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Deployment_CanarySetting) AWSCloudFormationType() string {
	return "AWS::ApiGateway::Deployment.CanarySetting"
}
