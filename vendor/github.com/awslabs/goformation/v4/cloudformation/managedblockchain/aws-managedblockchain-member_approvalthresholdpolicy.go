package managedblockchain

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Member_ApprovalThresholdPolicy AWS CloudFormation Resource (AWS::ManagedBlockchain::Member.ApprovalThresholdPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-approvalthresholdpolicy.html
type Member_ApprovalThresholdPolicy struct {

	// ProposalDurationInHours AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-approvalthresholdpolicy.html#cfn-managedblockchain-member-approvalthresholdpolicy-proposaldurationinhours
	ProposalDurationInHours int `json:"ProposalDurationInHours,omitempty"`

	// ThresholdComparator AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-approvalthresholdpolicy.html#cfn-managedblockchain-member-approvalthresholdpolicy-thresholdcomparator
	ThresholdComparator string `json:"ThresholdComparator,omitempty"`

	// ThresholdPercentage AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-managedblockchain-member-approvalthresholdpolicy.html#cfn-managedblockchain-member-approvalthresholdpolicy-thresholdpercentage
	ThresholdPercentage int `json:"ThresholdPercentage,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Member_ApprovalThresholdPolicy) AWSCloudFormationType() string {
	return "AWS::ManagedBlockchain::Member.ApprovalThresholdPolicy"
}
