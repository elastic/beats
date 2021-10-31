package codedeploy

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DeploymentGroup_OnPremisesTagSet AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.OnPremisesTagSet)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-onpremisestagset.html
type DeploymentGroup_OnPremisesTagSet struct {

	// OnPremisesTagSetList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-onpremisestagset.html#cfn-codedeploy-deploymentgroup-onpremisestagset-onpremisestagsetlist
	OnPremisesTagSetList []DeploymentGroup_OnPremisesTagSetListObject `json:"OnPremisesTagSetList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DeploymentGroup_OnPremisesTagSet) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.OnPremisesTagSet"
}
