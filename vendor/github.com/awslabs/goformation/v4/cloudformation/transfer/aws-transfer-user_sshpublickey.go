package transfer

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// User_SshPublicKey AWS CloudFormation Resource (AWS::Transfer::User.SshPublicKey)
// See:
type User_SshPublicKey struct {

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *User_SshPublicKey) AWSCloudFormationType() string {
	return "AWS::Transfer::User.SshPublicKey"
}
