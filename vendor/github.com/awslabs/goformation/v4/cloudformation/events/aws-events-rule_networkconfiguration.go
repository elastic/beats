package events

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Rule_NetworkConfiguration AWS CloudFormation Resource (AWS::Events::Rule.NetworkConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-networkconfiguration.html
type Rule_NetworkConfiguration struct {

	// AwsVpcConfiguration AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-events-rule-networkconfiguration.html#cfn-events-rule-networkconfiguration-awsvpcconfiguration
	AwsVpcConfiguration *Rule_AwsVpcConfiguration `json:"AwsVpcConfiguration,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Rule_NetworkConfiguration) AWSCloudFormationType() string {
	return "AWS::Events::Rule.NetworkConfiguration"
}
