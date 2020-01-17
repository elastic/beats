package batch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// JobDefinition_RetryStrategy AWS CloudFormation Resource (AWS::Batch::JobDefinition.RetryStrategy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-retrystrategy.html
type JobDefinition_RetryStrategy struct {

	// Attempts AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-batch-jobdefinition-retrystrategy.html#cfn-batch-jobdefinition-retrystrategy-attempts
	Attempts int `json:"Attempts,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *JobDefinition_RetryStrategy) AWSCloudFormationType() string {
	return "AWS::Batch::JobDefinition.RetryStrategy"
}
