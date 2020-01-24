package codedeploy

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeploymentGroup_EC2TagSetListObject AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.EC2TagSetListObject)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-ec2tagsetlistobject.html
type DeploymentGroup_EC2TagSetListObject struct {

	// Ec2TagGroup AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-ec2tagsetlistobject.html#cfn-codedeploy-deploymentgroup-ec2tagsetlistobject-ec2taggroup
	Ec2TagGroup []DeploymentGroup_EC2TagFilter `json:"Ec2TagGroup,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeploymentGroup_EC2TagSetListObject) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.EC2TagSetListObject"
}
