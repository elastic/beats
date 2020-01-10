package serverless

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Function_DeploymentPreference AWS CloudFormation Resource (AWS::Serverless::Function.DeploymentPreference)
// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
type Function_DeploymentPreference struct {

	// Alarms AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
	Alarms []string `json:"Alarms,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
	Enabled bool `json:"Enabled,omitempty"`

	// Hooks AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
	Hooks map[string]string `json:"Hooks,omitempty"`

	// Role AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
	Role string `json:"Role,omitempty"`

	// Type AWS CloudFormation Property
	// Required: false
	// See: https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md#deploymentpreference-object
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Function_DeploymentPreference) AWSCloudFormationType() string {
	return "AWS::Serverless::Function.DeploymentPreference"
}
