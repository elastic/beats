package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AccessPoint_PublicAccessBlockConfiguration AWS CloudFormation Resource (AWS::S3::AccessPoint.PublicAccessBlockConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-publicaccessblockconfiguration.html
type AccessPoint_PublicAccessBlockConfiguration struct {

	// BlockPublicAcls AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-publicaccessblockconfiguration.html#cfn-s3-accesspoint-publicaccessblockconfiguration-blockpublicacls
	BlockPublicAcls bool `json:"BlockPublicAcls,omitempty"`

	// BlockPublicPolicy AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-publicaccessblockconfiguration.html#cfn-s3-accesspoint-publicaccessblockconfiguration-blockpublicpolicy
	BlockPublicPolicy bool `json:"BlockPublicPolicy,omitempty"`

	// IgnorePublicAcls AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-publicaccessblockconfiguration.html#cfn-s3-accesspoint-publicaccessblockconfiguration-ignorepublicacls
	IgnorePublicAcls bool `json:"IgnorePublicAcls,omitempty"`

	// RestrictPublicBuckets AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-publicaccessblockconfiguration.html#cfn-s3-accesspoint-publicaccessblockconfiguration-restrictpublicbuckets
	RestrictPublicBuckets bool `json:"RestrictPublicBuckets,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AccessPoint_PublicAccessBlockConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::AccessPoint.PublicAccessBlockConfiguration"
}
