package cloudformation

// AWSEC2NetworkAclEntry_PortRange AWS CloudFormation Resource (AWS::EC2::NetworkAclEntry.PortRange)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-portrange.html
type AWSEC2NetworkAclEntry_PortRange struct {

	// From AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-portrange.html#cfn-ec2-networkaclentry-portrange-from
	From int `json:"From,omitempty"`

	// To AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-portrange.html#cfn-ec2-networkaclentry-portrange-to
	To int `json:"To,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2NetworkAclEntry_PortRange) AWSCloudFormationType() string {
	return "AWS::EC2::NetworkAclEntry.PortRange"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2NetworkAclEntry_PortRange) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
