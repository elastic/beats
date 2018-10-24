package cloudformation

// AWSEC2LaunchTemplate_CreditSpecification AWS CloudFormation Resource (AWS::EC2::LaunchTemplate.CreditSpecification)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplatedata-creditspecification.html
type AWSEC2LaunchTemplate_CreditSpecification struct {

	// CpuCredits AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ec2-launchtemplate-launchtemplatedata-creditspecification.html#cfn-ec2-launchtemplate-launchtemplatedata-creditspecification-cpucredits
	CpuCredits string `json:"CpuCredits,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSEC2LaunchTemplate_CreditSpecification) AWSCloudFormationType() string {
	return "AWS::EC2::LaunchTemplate.CreditSpecification"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSEC2LaunchTemplate_CreditSpecification) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
