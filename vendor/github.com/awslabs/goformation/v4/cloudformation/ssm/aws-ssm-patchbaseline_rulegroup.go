package ssm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// PatchBaseline_RuleGroup AWS CloudFormation Resource (AWS::SSM::PatchBaseline.RuleGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rulegroup.html
type PatchBaseline_RuleGroup struct {

	// PatchRules AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-rulegroup.html#cfn-ssm-patchbaseline-rulegroup-patchrules
	PatchRules []PatchBaseline_Rule `json:"PatchRules,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *PatchBaseline_RuleGroup) AWSCloudFormationType() string {
	return "AWS::SSM::PatchBaseline.RuleGroup"
}
