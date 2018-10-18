package cloudformation

// AWSSSMPatchBaseline_PatchFilterGroup AWS CloudFormation Resource (AWS::SSM::PatchBaseline.PatchFilterGroup)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-patchfiltergroup.html
type AWSSSMPatchBaseline_PatchFilterGroup struct {

	// PatchFilters AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-ssm-patchbaseline-patchfiltergroup.html#cfn-ssm-patchbaseline-patchfiltergroup-patchfilters
	PatchFilters []AWSSSMPatchBaseline_PatchFilter `json:"PatchFilters,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMPatchBaseline_PatchFilterGroup) AWSCloudFormationType() string {
	return "AWS::SSM::PatchBaseline.PatchFilterGroup"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSSMPatchBaseline_PatchFilterGroup) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
