package cloudformation

// AWSEC2NetworkAclEntry_Icmp AWS CloudFormation Resource (AWS::EC2::NetworkAclEntry.Icmp)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-icmp.html
type AWSEC2NetworkAclEntry_Icmp struct {

	// Code AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-icmp.html#cfn-ec2-networkaclentry-icmp-code
	Code int `json:"Code,omitempty"`

	// Type AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-networkaclentry-icmp.html#cfn-ec2-networkaclentry-icmp-type
	Type int `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2NetworkAclEntry_Icmp) AWSCloudFormationType() string {
	return "AWS::EC2::NetworkAclEntry.Icmp"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2NetworkAclEntry_Icmp) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
