package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SecurityConfiguration_EncryptionConfiguration AWS CloudFormation Resource (AWS::Glue::SecurityConfiguration.EncryptionConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-encryptionconfiguration.html
type SecurityConfiguration_EncryptionConfiguration struct {

	// CloudWatchEncryption AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-encryptionconfiguration.html#cfn-glue-securityconfiguration-encryptionconfiguration-cloudwatchencryption
	CloudWatchEncryption *SecurityConfiguration_CloudWatchEncryption `json:"CloudWatchEncryption,omitempty"`

	// JobBookmarksEncryption AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-encryptionconfiguration.html#cfn-glue-securityconfiguration-encryptionconfiguration-jobbookmarksencryption
	JobBookmarksEncryption *SecurityConfiguration_JobBookmarksEncryption `json:"JobBookmarksEncryption,omitempty"`

	// S3Encryptions AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-encryptionconfiguration.html#cfn-glue-securityconfiguration-encryptionconfiguration-s3encryptions
	S3Encryptions *SecurityConfiguration_S3Encryptions `json:"S3Encryptions,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SecurityConfiguration_EncryptionConfiguration) AWSCloudFormationType() string {
	return "AWS::Glue::SecurityConfiguration.EncryptionConfiguration"
}
