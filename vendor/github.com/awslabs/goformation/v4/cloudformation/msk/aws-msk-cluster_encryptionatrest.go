package msk

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_EncryptionAtRest AWS CloudFormation Resource (AWS::MSK::Cluster.EncryptionAtRest)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-encryptionatrest.html
type Cluster_EncryptionAtRest struct {

	// DataVolumeKMSKeyId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-encryptionatrest.html#cfn-msk-cluster-encryptionatrest-datavolumekmskeyid
	DataVolumeKMSKeyId string `json:"DataVolumeKMSKeyId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_EncryptionAtRest) AWSCloudFormationType() string {
	return "AWS::MSK::Cluster.EncryptionAtRest"
}
