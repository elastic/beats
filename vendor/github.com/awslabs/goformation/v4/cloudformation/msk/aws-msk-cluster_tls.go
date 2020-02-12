package msk

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Cluster_Tls AWS CloudFormation Resource (AWS::MSK::Cluster.Tls)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-tls.html
type Cluster_Tls struct {

	// CertificateAuthorityArnList AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-msk-cluster-tls.html#cfn-msk-cluster-tls-certificateauthorityarnlist
	CertificateAuthorityArnList []string `json:"CertificateAuthorityArnList,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Cluster_Tls) AWSCloudFormationType() string {
	return "AWS::MSK::Cluster.Tls"
}
