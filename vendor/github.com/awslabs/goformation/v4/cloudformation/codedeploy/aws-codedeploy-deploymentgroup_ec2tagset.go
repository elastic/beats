package codedeploy

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeploymentGroup_EC2TagSet AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.EC2TagSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-ec2tagset.html
type DeploymentGroup_EC2TagSet struct {

	// Ec2TagSetList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-ec2tagset.html#cfn-codedeploy-deploymentgroup-ec2tagset-ec2tagsetlist
	Ec2TagSetList []DeploymentGroup_EC2TagSetListObject `json:"Ec2TagSetList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeploymentGroup_EC2TagSet) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.EC2TagSet"
}
