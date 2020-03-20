package codepipeline

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Pipeline_EncryptionKey AWS CloudFormation Resource (AWS::CodePipeline::Pipeline.EncryptionKey)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-artifactstore-encryptionkey.html
type Pipeline_EncryptionKey struct {

	// Id AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-artifactstore-encryptionkey.html#cfn-codepipeline-pipeline-artifactstore-encryptionkey-id
	Id string `json:"Id,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-artifactstore-encryptionkey.html#cfn-codepipeline-pipeline-artifactstore-encryptionkey-type
	Type string `json:"Type,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Pipeline_EncryptionKey) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Pipeline.EncryptionKey"
}
