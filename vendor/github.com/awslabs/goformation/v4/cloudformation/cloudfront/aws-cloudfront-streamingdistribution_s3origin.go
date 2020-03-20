package cloudfront

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// StreamingDistribution_S3Origin AWS CloudFormation Resource (AWS::CloudFront::StreamingDistribution.S3Origin)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html
type StreamingDistribution_S3Origin struct {

	// DomainName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html#cfn-cloudfront-streamingdistribution-s3origin-domainname
	DomainName string `json:"DomainName,omitempty"`

	// OriginAccessIdentity AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-cloudfront-streamingdistribution-s3origin.html#cfn-cloudfront-streamingdistribution-s3origin-originaccessidentity
	OriginAccessIdentity string `json:"OriginAccessIdentity,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *StreamingDistribution_S3Origin) AWSCloudFormationType() string {
	return "AWS::CloudFront::StreamingDistribution.S3Origin"
}
