package apigateway

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Deployment_DeploymentCanarySettings AWS CloudFormation Resource (AWS::ApiGateway::Deployment.DeploymentCanarySettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-deploymentcanarysettings.html
type Deployment_DeploymentCanarySettings struct {

	// PercentTraffic AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-deploymentcanarysettings.html#cfn-apigateway-deployment-deploymentcanarysettings-percenttraffic
	PercentTraffic float64 `json:"PercentTraffic,omitempty"`

	// StageVariableOverrides AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-deploymentcanarysettings.html#cfn-apigateway-deployment-deploymentcanarysettings-stagevariableoverrides
	StageVariableOverrides map[string]string `json:"StageVariableOverrides,omitempty"`

	// UseStageCache AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-apigateway-deployment-deploymentcanarysettings.html#cfn-apigateway-deployment-deploymentcanarysettings-usestagecache
	UseStageCache bool `json:"UseStageCache,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Deployment_DeploymentCanarySettings) AWSCloudFormationType() string {
	return "AWS::ApiGateway::Deployment.DeploymentCanarySettings"
}
