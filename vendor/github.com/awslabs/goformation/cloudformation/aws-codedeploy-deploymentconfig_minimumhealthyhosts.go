package cloudformation

// AWSCodeDeployDeploymentConfig_MinimumHealthyHosts AWS CloudFormation Resource (AWS::CodeDeploy::DeploymentConfig.MinimumHealthyHosts)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentconfig-minimumhealthyhosts.html
type AWSCodeDeployDeploymentConfig_MinimumHealthyHosts struct {

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentconfig-minimumhealthyhosts.html#cfn-codedeploy-deploymentconfig-minimumhealthyhosts-type
	Type string `json:"Type,omitempty"`

	// Value AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codedeploy-deploymentconfig-minimumhealthyhosts.html#cfn-codedeploy-deploymentconfig-minimumhealthyhosts-value
	Value int `json:"Value,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeDeployDeploymentConfig_MinimumHealthyHosts) AWSCloudFormationType() string {
	return "AWS::CodeDeploy::DeploymentConfig.MinimumHealthyHosts"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeDeployDeploymentConfig_MinimumHealthyHosts) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
