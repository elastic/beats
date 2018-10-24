package cloudformation

// AWSWAFXssMatchSet_XssMatchTuple AWS CloudFormation Resource (AWS::WAF::XssMatchSet.XssMatchTuple)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-xssmatchset-xssmatchtuple.html
type AWSWAFXssMatchSet_XssMatchTuple struct {

	// FieldToMatch AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-xssmatchset-xssmatchtuple.html#cfn-waf-xssmatchset-xssmatchtuple-fieldtomatch
	FieldToMatch *AWSWAFXssMatchSet_FieldToMatch `json:"FieldToMatch,omitempty"`

	// TextTransformation AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-xssmatchset-xssmatchtuple.html#cfn-waf-xssmatchset-xssmatchtuple-texttransformation
	TextTransformation string `json:"TextTransformation,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFXssMatchSet_XssMatchTuple) AWSCloudFormationType() string {
	return "AWS::WAF::XssMatchSet.XssMatchTuple"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFXssMatchSet_XssMatchTuple) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
