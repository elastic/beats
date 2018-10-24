package cloudformation

// AWSBudgetsBudget_Spend AWS CloudFormation Resource (AWS::Budgets::Budget.Spend)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-spend.html
type AWSBudgetsBudget_Spend struct {

	// Amount AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-spend.html#cfn-budgets-budget-spend-amount
	Amount float64 `json:"Amount,omitempty"`

	// Unit AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-budgets-budget-spend.html#cfn-budgets-budget-spend-unit
	Unit string `json:"Unit,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSBudgetsBudget_Spend) AWSCloudFormationType() string {
	return "AWS::Budgets::Budget.Spend"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSBudgetsBudget_Spend) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
