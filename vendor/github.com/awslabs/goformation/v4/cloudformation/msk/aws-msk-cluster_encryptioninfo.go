package msk

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_EncryptionInfo AWS CloudFormation Resource (AWS::MSK::Cluster.EncryptionInfo)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-encryptioninfo.html
type Cluster_EncryptionInfo struct {

	// EncryptionAtRest AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-encryptioninfo.html#cfn-msk-cluster-encryptioninfo-encryptionatrest
	EncryptionAtRest *Cluster_EncryptionAtRest `json:"EncryptionAtRest,omitempty"`

	// EncryptionInTransit AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-encryptioninfo.html#cfn-msk-cluster-encryptioninfo-encryptionintransit
	EncryptionInTransit *Cluster_EncryptionInTransit `json:"EncryptionInTransit,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_EncryptionInfo) AWSCloudFormationType() string {
	return "AWS::MSK::Cluster.EncryptionInfo"
}
