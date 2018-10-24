package cloudformation

// AWSSSMAssociation_Target AWS CloudFormation Resource (AWS::SSM::Association.Target)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-target.html
type AWSSSMAssociation_Target struct {

	// Key AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-target.html#cfn-ssm-association-target-key
	Key string `json:"Key,omitempty"`

	// Values AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-target.html#cfn-ssm-association-target-values
	Values []string `json:"Values,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMAssociation_Target) AWSCloudFormationType() string {
	return "AWS::SSM::Association.Target"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMAssociation_Target) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
