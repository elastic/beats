package dlm

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// LifecyclePolicy_Parameters AWS CloudFormation Resource (AWS::DLM::LifecyclePolicy.Parameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dlm-lifecyclepolicy-parameters.html
type LifecyclePolicy_Parameters struct {

	// ExcludeBootVolume AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-dlm-lifecyclepolicy-parameters.html#cfn-dlm-lifecyclepolicy-parameters-excludebootvolume
	ExcludeBootVolume bool `json:"ExcludeBootVolume,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *LifecyclePolicy_Parameters) AWSCloudFormationType() string {
	return "AWS::DLM::LifecyclePolicy.Parameters"
}
