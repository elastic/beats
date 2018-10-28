package cloudformation

// AWSEC2LaunchTemplate_ElasticGpuSpecification AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.ElasticGpuSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-elasticgpuspecification.html
type AWSEC2LaunchTemplate_ElasticGpuSpecification struct {

	// Type AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-elasticgpuspecification.html#cfn-ec2-launchtemplate-elasticgpuspecification-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2LaunchTemplate_ElasticGpuSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.ElasticGpuSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2LaunchTemplate_ElasticGpuSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
