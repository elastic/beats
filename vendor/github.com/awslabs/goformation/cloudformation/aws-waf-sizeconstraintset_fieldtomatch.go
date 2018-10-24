package cloudformation

// AWSWAFSizeConstraintSet_FieldToMatch AWS CloudFormation Resource (AWS::WAF::SizeConstraintSet.FieldToMatch)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint-fieldtomatch.html
type AWSWAFSizeConstraintSet_FieldToMatch struct {

	// Data AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint-fieldtomatch.html#cfn-waf-sizeconstraintset-sizeconstraint-fieldtomatch-data
	Data string `json:"Data,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint-fieldtomatch.html#cfn-waf-sizeconstraintset-sizeconstraint-fieldtomatch-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFSizeConstraintSet_FieldToMatch) AWSCloudFormationType() string {
	return "AWS::WAF::SizeConstraintSet.FieldToMatch"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFSizeConstraintSet_FieldToMatch) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
