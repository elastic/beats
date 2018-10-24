package cloudformation

// AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook AWS CloudFormation Resource (AWS::SageMaker::NotebookInstanceLifecycleConfig.NotebookInstanceLifecycleHook)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-notebookinstancelifecycleconfig-notebookinstancelifecyclehook.html
type AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook struct {

	// Content AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-sagemaker-notebookinstancelifecycleconfig-notebookinstancelifecyclehook.html#cfn-sagemaker-notebookinstancelifecycleconfig-notebookinstancelifecyclehook-content
	Content string `json:"Content,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook) AWSCloudFormationType() string {
	return "AWS::SageMaker::NotebookInstanceLifecycleConfig.NotebookInstanceLifecycleHook"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSSageMakerNotebookInstanceLifecycleConfig_NotebookInstanceLifecycleHook) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
