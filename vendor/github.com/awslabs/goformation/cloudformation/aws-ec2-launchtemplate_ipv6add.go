package cloudformation

// AWSEC2LaunchTemplate_Ipv6Add AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.Ipv6Add)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-ipv6add.html
type AWSEC2LaunchTemplate_Ipv6Add struct {

	// Ipv6Address AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-ipv6add.html#cfn-ec2-launchtemplate-ipv6add-ipv6address
	Ipv6Address string `json:"Ipv6Address,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2LaunchTemplate_Ipv6Add) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.Ipv6Add"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2LaunchTemplate_Ipv6Add) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
