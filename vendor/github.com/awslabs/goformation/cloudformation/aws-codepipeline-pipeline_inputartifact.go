package cloudformation

// AWSCodePipelinePipeline_InputArtifact AWS CloudFormation Resource (AWS::CodePipeline::Pipeline.InputArtifact)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages-actions-inputartifacts.html
type AWSCodePipelinePipeline_InputArtifact struct {

	// Name AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codepipeline-pipeline-stages-actions-inputartifacts.html#cfn-codepipeline-pipeline-stages-actions-inputartifacts-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodePipelinePipeline_InputArtifact) AWSCloudFormationType() string {
	return "AWS::CodePipeline::Pipeline.InputArtifact"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodePipelinePipeline_InputArtifact) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
