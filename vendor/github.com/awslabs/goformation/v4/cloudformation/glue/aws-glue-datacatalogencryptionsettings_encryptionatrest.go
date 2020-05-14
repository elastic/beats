package glue

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// DataCatalogEncryptionSettings_EncryptionAtRest AWS CloudFormation Resource (AWS::Glue::DataCatalogEncryptionSettings.EncryptionAtRest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-encryptionatrest.html
type DataCatalogEncryptionSettings_EncryptionAtRest struct {

	// CatalogEncryptionMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-encryptionatrest.html#cfn-glue-datacatalogencryptionsettings-encryptionatrest-catalogencryptionmode
	CatalogEncryptionMode string `json:"CatalogEncryptionMode,omitempty"`

	// SseAwsKmsKeyId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-glue-datacatalogencryptionsettings-encryptionatrest.html#cfn-glue-datacatalogencryptionsettings-encryptionatrest-sseawskmskeyid
	SseAwsKmsKeyId string `json:"SseAwsKmsKeyId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *DataCatalogEncryptionSettings_EncryptionAtRest) AWSCloudFormationType() string {
	return "AWS::Glue::DataCatalogEncryptionSettings.EncryptionAtRest"
}
