package lambda

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Version_ProvisionedConcurrencyConfiguration AWS CloudFormation Resource (AWS::Lambda::Version.ProvisionedConcurrencyConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-version-provisionedconcurrencyconfiguration.html
type Version_ProvisionedConcurrencyConfiguration struct {

	// ProvisionedConcurrentExecutions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-lambda-version-provisionedconcurrencyconfiguration.html#cfn-lambda-version-provisionedconcurrencyconfiguration-provisionedconcurrentexecutions
	ProvisionedConcurrentExecutions int `json:"ProvisionedConcurrentExecutions"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Version_ProvisionedConcurrencyConfiguration) AWSCloudFormationType() string {
	return "AWS::Lambda::Version.ProvisionedConcurrencyConfiguration"
}
