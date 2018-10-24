package cloudformation

// AWSSSMPatchBaseline_PatchGroup AWS CloudFormation Resource (AWS::SSM::PatchBaseline.PatchGroup)
// See:
type AWSSSMPatchBaseline_PatchGroup struct {
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSSMPatchBaseline_PatchGroup) AWSCloudFormationType() string {
	return "AWS::SSM::PatchBaseline.PatchGroup"
}
