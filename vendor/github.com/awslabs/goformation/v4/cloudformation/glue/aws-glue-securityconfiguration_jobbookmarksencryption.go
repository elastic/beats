package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SecurityConfiguration_JobBookmarksEncryption AWS CloudFormation Resource (AWS::Glue::SecurityConfiguration.JobBookmarksEncryption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-jobbookmarksencryption.html
type SecurityConfiguration_JobBookmarksEncryption struct {

	// JobBookmarksEncryptionMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-jobbookmarksencryption.html#cfn-glue-securityconfiguration-jobbookmarksencryption-jobbookmarksencryptionmode
	JobBookmarksEncryptionMode string `json:"JobBookmarksEncryptionMode,omitempty"`

	// KmsKeyArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-jobbookmarksencryption.html#cfn-glue-securityconfiguration-jobbookmarksencryption-kmskeyarn
	KmsKeyArn string `json:"KmsKeyArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SecurityConfiguration_JobBookmarksEncryption) AWSCloudFormationType() string {
	return "AWS::Glue::SecurityConfiguration.JobBookmarksEncryption"
}
