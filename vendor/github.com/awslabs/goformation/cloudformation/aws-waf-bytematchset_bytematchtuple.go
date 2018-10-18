package cloudformation

// AWSWAFByteMatchSet_ByteMatchTuple AWS CloudFormation Resource (AWS::WAF::ByteMatchSet.ByteMatchTuple)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html
type AWSWAFByteMatchSet_ByteMatchTuple struct {

	// FieldToMatch AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html#cfn-waf-bytematchset-bytematchtuples-fieldtomatch
	FieldToMatch *AWSWAFByteMatchSet_FieldToMatch `json:"FieldToMatch,omitempty"`

	// PositionalConstraint AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html#cfn-waf-bytematchset-bytematchtuples-positionalconstraint
	PositionalConstraint string `json:"PositionalConstraint,omitempty"`

	// TargetString AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html#cfn-waf-bytematchset-bytematchtuples-targetstring
	TargetString string `json:"TargetString,omitempty"`

	// TargetStringBase64 AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html#cfn-waf-bytematchset-bytematchtuples-targetstringbase64
	TargetStringBase64 string `json:"TargetStringBase64,omitempty"`

	// TextTransformation AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-bytematchset-bytematchtuples.html#cfn-waf-bytematchset-bytematchtuples-texttransformation
	TextTransformation string `json:"TextTransformation,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFByteMatchSet_ByteMatchTuple) AWSCloudFormationType() string {
	return "AWS::WAF::ByteMatchSet.ByteMatchTuple"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFByteMatchSet_ByteMatchTuple) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
