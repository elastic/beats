package cloudformation

// AWSSSMPatchBaseline_Rule AWS CloudFormation Resource (AWS::SSM::PatchBaseline.Rule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html
type AWSSSMPatchBaseline_Rule struct {

	// ApproveAfterDays AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html#cfn-ssm-patchbaseline-rule-approveafterdays
	ApproveAfterDays int `json:"ApproveAfterDays,omitempty"`

	// ComplianceLevel AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html#cfn-ssm-patchbaseline-rule-compliancelevel
	ComplianceLevel string `json:"ComplianceLevel,omitempty"`

	// EnableNonSecurity AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html#cfn-ssm-patchbaseline-rule-enablenonsecurity
	EnableNonSecurity bool `json:"EnableNonSecurity,omitempty"`

	// PatchFilterGroup AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html#cfn-ssm-patchbaseline-rule-patchfiltergroup
	PatchFilterGroup *AWSSSMPatchBaseline_PatchFilterGroup `json:"PatchFilterGroup,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMPatchBaseline_Rule) AWSCloudFormationType() string {
	return "AWS::SSM::PatchBaseline.Rule"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMPatchBaseline_Rule) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
