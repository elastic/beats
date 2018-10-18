package cloudformation

// AWSGuardDutyFilter_FindingCriteria AWS CloudFormation Resource (AWS::GuardDuty::Filter.FindingCriteria)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-guardduty-filter-findingcriteria.html
type AWSGuardDutyFilter_FindingCriteria struct {

	// Criterion AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-guardduty-filter-findingcriteria.html#cfn-guardduty-filter-findingcriteria-criterion
	Criterion interface{} `json:"Criterion,omitempty"`

	// ItemType AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-guardduty-filter-findingcriteria.html#cfn-guardduty-filter-findingcriteria-itemtype
	ItemType *AWSGuardDutyFilter_Condition `json:"ItemType,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSGuardDutyFilter_FindingCriteria) AWSCloudFormationType() string {
	return "AWS::GuardDuty::Filter.FindingCriteria"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSGuardDutyFilter_FindingCriteria) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
