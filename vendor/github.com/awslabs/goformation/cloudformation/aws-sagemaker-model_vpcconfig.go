package cloudformation

// AWSSageMakerModel_VpcConfig AWS CloudFormation Resource (AWS::SageMaker::Model.VpcConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-model-vpcconfig.html
type AWSSageMakerModel_VpcConfig struct {

	// SecurityGroupIds AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-model-vpcconfig.html#cfn-sagemaker-model-vpcconfig-securitygroupids
	SecurityGroupIds []string `json:"SecurityGroupIds,omitempty"`

	// Subnets AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-model-vpcconfig.html#cfn-sagemaker-model-vpcconfig-subnets
	Subnets []string `json:"Subnets,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSageMakerModel_VpcConfig) AWSCloudFormationType() string {
	return "AWS::SageMaker::Model.VpcConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSageMakerModel_VpcConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
