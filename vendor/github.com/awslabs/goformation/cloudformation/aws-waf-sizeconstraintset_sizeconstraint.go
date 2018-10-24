package cloudformation

// AWSWAFSizeConstraintSet_SizeConstraint AWS CloudFormation Resource (AWS::WAF::SizeConstraintSet.SizeConstraint)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint.html
type AWSWAFSizeConstraintSet_SizeConstraint struct {

	// ComparisonOperator AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint.html#cfn-waf-sizeconstraintset-sizeconstraint-comparisonoperator
	ComparisonOperator string `json:"ComparisonOperator,omitempty"`

	// FieldToMatch AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint.html#cfn-waf-sizeconstraintset-sizeconstraint-fieldtomatch
	FieldToMatch *AWSWAFSizeConstraintSet_FieldToMatch `json:"FieldToMatch,omitempty"`

	// Size AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint.html#cfn-waf-sizeconstraintset-sizeconstraint-size
	Size int `json:"Size,omitempty"`

	// TextTransformation AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-waf-sizeconstraintset-sizeconstraint.html#cfn-waf-sizeconstraintset-sizeconstraint-texttransformation
	TextTransformation string `json:"TextTransformation,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSWAFSizeConstraintSet_SizeConstraint) AWSCloudFormationType() string {
	return "AWS::WAF::SizeConstraintSet.SizeConstraint"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSWAFSizeConstraintSet_SizeConstraint) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
