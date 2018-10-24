package cloudformation

// AWSEC2LaunchTemplate_TagSpecification AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.TagSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-tagspecification.html
type AWSEC2LaunchTemplate_TagSpecification struct {

	// ResourceType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-tagspecification.html#cfn-ec2-launchtemplate-tagspecification-resourcetype
	ResourceType string `json:"ResourceType,omitempty"`

	// Tags AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-tagspecification.html#cfn-ec2-launchtemplate-tagspecification-tags
	Tags []Tag `json:"Tags,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2LaunchTemplate_TagSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.TagSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2LaunchTemplate_TagSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
