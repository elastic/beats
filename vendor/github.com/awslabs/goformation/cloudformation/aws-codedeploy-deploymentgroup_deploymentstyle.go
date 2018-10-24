package cloudformation

// AWSCodeDeployDeploymentGroup_DeploymentStyle AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentGroup.DeploymentStyle)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deploymentstyle.html
type AWSCodeDeployDeploymentGroup_DeploymentStyle struct {

	// DeploymentOption AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deploymentstyle.html#cfn-codedeploy-deploymentgroup-deploymentstyle-deploymentoption
	DeploymentOption string `json:"DeploymentOption,omitempty"`

	// DeploymentType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentgroup-deploymentstyle.html#cfn-codedeploy-deploymentgroup-deploymentstyle-deploymenttype
	DeploymentType string `json:"DeploymentType,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentGroup_DeploymentStyle) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentGroup.DeploymentStyle"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentGroup_DeploymentStyle) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
