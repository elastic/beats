package cloudformation

// AWSEC2Instance_NoDevice AWS CloudFormation Resource (AWS::EC2::Instance.NoDevice)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-instance-nodevice.html
type AWSEC2Instance_NoDevice struct {

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2Instance_NoDevice) AWSCloudFormationType() string {
	return "AWS::EC2::Instance.NoDevice"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2Instance_NoDevice) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
