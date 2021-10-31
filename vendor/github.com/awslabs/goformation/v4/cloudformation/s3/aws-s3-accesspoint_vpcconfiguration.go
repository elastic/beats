package s3

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// AccessPoint_VpcConfiguration AWS CloudFormation Resource (AWS::S3::AccessPoint.VpcConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-vpcconfiguration.html
type AccessPoint_VpcConfiguration struct {

	// VpcId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-accesspoint-vpcconfiguration.html#cfn-s3-accesspoint-vpcconfiguration-vpcid
	VpcId string `json:"VpcId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AccessPoint_VpcConfiguration) AWSCloudFormationType() string {
	return "AWS::S3::AccessPoint.VpcConfiguration"
}
