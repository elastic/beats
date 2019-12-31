package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// SecurityConfiguration_CloudWatchEncryption AWS CloudFormation Resource (AWS::Glue::SecurityConfiguration.CloudWatchEncryption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-cloudwatchencryption.html
type SecurityConfiguration_CloudWatchEncryption struct {

	// CloudWatchEncryptionMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-cloudwatchencryption.html#cfn-glue-securityconfiguration-cloudwatchencryption-cloudwatchencryptionmode
	CloudWatchEncryptionMode string `json:"CloudWatchEncryptionMode,omitempty"`

	// KmsKeyArn AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-securityconfiguration-cloudwatchencryption.html#cfn-glue-securityconfiguration-cloudwatchencryption-kmskeyarn
	KmsKeyArn string `json:"KmsKeyArn,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *SecurityConfiguration_CloudWatchEncryption) AWSCloudFormationType() string {
	return "AWS::Glue::SecurityConfiguration.CloudWatchEncryption"
}
