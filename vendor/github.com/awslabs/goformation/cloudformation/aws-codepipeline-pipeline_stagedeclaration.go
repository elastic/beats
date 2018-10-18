package cloudformation

// AWSCodePipelinePipeline_StageDeclaration AWS CloudFormation Resource (AWS::CodePipeline::Pipeline.StageDeclaration)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages.html
type AWSCodePipelinePipeline_StageDeclaration struct {

	// Actions AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages.html#cfn-codepipeline-pipeline-stages-actions
	Actions []AWSCodePipelinePipeline_ActionDeclaration `json:"Actions,omitempty"`

	// Blockers AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages.html#cfn-codepipeline-pipeline-stages-blockers
	Blockers []AWSCodePipelinePipeline_BlockerDeclaration `json:"Blockers,omitempty"`

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages.html#cfn-codepipeline-pipeline-stages-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodePipelinePipeline_StageDeclaration) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Pipeline.StageDeclaration"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodePipelinePipeline_StageDeclaration) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
