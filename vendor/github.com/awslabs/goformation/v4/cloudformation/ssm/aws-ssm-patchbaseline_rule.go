package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// PatchBaseline_Rule AWS CloudFormation Resource (AWS::SSM::PatchBaseline.Rule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rule.html
type PatchBaseline_Rule struct {

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
	PatchFilterGroup *PatchBaseline_PatchFilterGroup `json:"PatchFilterGroup,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *PatchBaseline_Rule) AWSCloudFormationType() string {
	return "AWS::SSM::PatchBaseline.Rule"
}
