package cloudformation

// AWSEC2SpotFleet_IamInstanceProfileSpecification AWS CloudFormation Resource (AWS::EC2::SpotFleet.IamInstanceProfileSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-spotfleetrequestconfigdata-launchspecifications-iaminstanceprofile.html
type AWSEC2SpotFleet_IamInstanceProfileSpecification struct {

	// Arn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-spotfleet-spotfleetrequestconfigdata-launchspecifications-iaminstanceprofile.html#cfn-ec2-spotfleet-iaminstanceprofilespecification-arn
	Arn string `json:"Arn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2SpotFleet_IamInstanceProfileSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::SpotFleet.IamInstanceProfileSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2SpotFleet_IamInstanceProfileSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
