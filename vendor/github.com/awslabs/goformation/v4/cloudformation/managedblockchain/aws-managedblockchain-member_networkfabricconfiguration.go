package managedblockchain

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Member_NetworkFabricConfiguration AWS CloudFormation Resource (AWS::ManagedBlockchain::Member.NetworkFabricConfiguration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-networkfabricconfiguration.html
type Member_NetworkFabricConfiguration struct {

	// Edition AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-networkfabricconfiguration.html#cfn-managedblockchain-member-networkfabricconfiguration-edition
	Edition string `json:"Edition,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Member_NetworkFabricConfiguration) AWSCloudFormationType() string {
	return "AWS::ManagedBlockchain::Member.NetworkFabricConfiguration"
}
