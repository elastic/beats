package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SecurityConfiguration_S3Encryptions AWS CloudFormation Resource (AWS::Glue::SecurityConfiguration.S3Encryptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-s3encryptions.html
type SecurityConfiguration_S3Encryptions struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SecurityConfiguration_S3Encryptions) AWSCloudFormationType() string {
	return "AWS::Glue::SecurityConfiguration.S3Encryptions"
}
