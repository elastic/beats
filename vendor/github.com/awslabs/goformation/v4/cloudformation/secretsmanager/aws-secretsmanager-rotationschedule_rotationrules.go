package secretsmanager

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// RotationSchedule_RotationRules AWS CloudFormation Resource (AWS::SecretsManager::RotationSchedule.RotationRules)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-secretsmanager-rotationschedule-rotationrules.html
type RotationSchedule_RotationRules struct {

	// AutomaticallyAfterDays AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-secretsmanager-rotationschedule-rotationrules.html#cfn-secretsmanager-rotationschedule-rotationrules-automaticallyafterdays
	AutomaticallyAfterDays int `json:"AutomaticallyAfterDays,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *RotationSchedule_RotationRules) AWSCloudFormationType() string {
	return "AWS::SecretsManager::RotationSchedule.RotationRules"
}
