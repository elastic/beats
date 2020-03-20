package dlm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LifecyclePolicy_RetainRule AWS CloudFormation Resource (AWS::DLM::LifecyclePolicy.RetainRule)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dlm-lifecyclepolicy-retainrule.html
type LifecyclePolicy_RetainRule struct {

	// Count AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dlm-lifecyclepolicy-retainrule.html#cfn-dlm-lifecyclepolicy-retainrule-count
	Count int `json:"Count"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LifecyclePolicy_RetainRule) AWSCloudFormationType() string {
	return "AWS::DLM::LifecyclePolicy.RetainRule"
}
