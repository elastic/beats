package cloudformation

// AWSBudgetsBudget_TimePeriod AWS CloudFormation Resource (AWS::Budgets::Budget.TimePeriod)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-timeperiod.html
type AWSBudgetsBudget_TimePeriod struct {

	// End AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-timeperiod.html#cfn-budgets-budget-timeperiod-end
	End string `json:"End,omitempty"`

	// Start AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-timeperiod.html#cfn-budgets-budget-timeperiod-start
	Start string `json:"Start,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSBudgetsBudget_TimePeriod) AWSCloudFormationType() string {
	return "AWS::Budgets::Budget.TimePeriod"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSBudgetsBudget_TimePeriod) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
