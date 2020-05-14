package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataCatalogEncryptionSettings_DataCatalogEncryptionSettings AWS CloudFormation Resource (AWS::Glue::DataCatalogEncryptionSettings.DataCatalogEncryptionSettings)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-datacatalogencryptionsettings.html
type DataCatalogEncryptionSettings_DataCatalogEncryptionSettings struct {

	// ConnectionPasswordEncryption AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-datacatalogencryptionsettings.html#cfn-glue-datacatalogencryptionsettings-datacatalogencryptionsettings-connectionpasswordencryption
	ConnectionPasswordEncryption *DataCatalogEncryptionSettings_ConnectionPasswordEncryption `json:"ConnectionPasswordEncryption,omitempty"`

	// EncryptionAtRest AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-datacatalogencryptionsettings.html#cfn-glue-datacatalogencryptionsettings-datacatalogencryptionsettings-encryptionatrest
	EncryptionAtRest *DataCatalogEncryptionSettings_EncryptionAtRest `json:"EncryptionAtRest,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataCatalogEncryptionSettings_DataCatalogEncryptionSettings) AWSCloudFormationType() string {
	return "AWS::Glue::DataCatalogEncryptionSettings.DataCatalogEncryptionSettings"
}
