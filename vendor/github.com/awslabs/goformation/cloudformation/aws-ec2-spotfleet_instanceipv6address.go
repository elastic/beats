package cloudformation

// AWSEC2SpotFleet_InstanceIpv6Address AWS CloudFormation Resource (AWS::EC2::SpotFleet.InstanceIpv6Address)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-instanceipv6address.html
type AWSEC2SpotFleet_InstanceIpv6Address struct {

	// Ipv6Address AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-instanceipv6address.html#cfn-ec2-spotfleet-instanceipv6address-ipv6address
	Ipv6Address string `json:"Ipv6Address,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2SpotFleet_InstanceIpv6Address) AWSCloudFormationType() string {
	return "AWS::EC2::SpotFleet.InstanceIpv6Address"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2SpotFleet_InstanceIpv6Address) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
