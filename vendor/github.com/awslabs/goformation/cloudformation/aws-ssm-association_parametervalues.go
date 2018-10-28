package cloudformation

// AWSSSMAssociation_ParameterValues AWS CloudFormation Resource (AWS::SSM::Association.ParameterValues)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-parametervalues.html
type AWSSSMAssociation_ParameterValues struct {

	// ParameterValues AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-association-parametervalues.html#cfn-ssm-association-parametervalues-parametervalues
	ParameterValues []string `json:"ParameterValues,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMAssociation_ParameterValues) AWSCloudFormationType() string {
	return "AWS::SSM::Association.ParameterValues"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMAssociation_ParameterValues) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
