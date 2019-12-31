package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataCatalogEncryptionSettings_ConnectionPasswordEncryption AWS CloudFormation Resource (AWS::Glue::DataCatalogEncryptionSettings.ConnectionPasswordEncryption)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-connectionpasswordencryption.html
type DataCatalogEncryptionSettings_ConnectionPasswordEncryption struct {

	// KmsKeyId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-connectionpasswordencryption.html#cfn-glue-datacatalogencryptionsettings-connectionpasswordencryption-kmskeyid
	KmsKeyId string `json:"KmsKeyId,omitempty"`

	// ReturnConnectionPasswordEncrypted AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-connectionpasswordencryption.html#cfn-glue-datacatalogencryptionsettings-connectionpasswordencryption-returnconnectionpasswordencrypted
	ReturnConnectionPasswordEncrypted bool `json:"ReturnConnectionPasswordEncrypted,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataCatalogEncryptionSettings_ConnectionPasswordEncryption) AWSCloudFormationType() string {
	return "AWS::Glue::DataCatalogEncryptionSettings.ConnectionPasswordEncryption"
}
