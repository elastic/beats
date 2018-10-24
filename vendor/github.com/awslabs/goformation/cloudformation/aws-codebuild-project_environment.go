package cloudformation

// AWSCodeBuildProject_Environment AWS CloudFormation Resource (AWS::CodeBuild::Project.Environment)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html
type AWSCodeBuildProject_Environment struct {

	// Certificate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-certificate
	Certificate string `json:"Certificate,omitempty"`

	// ComputeType AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-computetype
	ComputeType string `json:"ComputeType,omitempty"`

	// EnvironmentVariables AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-environmentvariables
	EnvironmentVariables []AWSCodeBuildProject_EnvironmentVariable `json:"EnvironmentVariables,omitempty"`

	// Image AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-image
	Image string `json:"Image,omitempty"`

	// PrivilegedMode AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-privilegedmode
	PrivilegedMode bool `json:"PrivilegedMode,omitempty"`

	// Type AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-codebuild-project-environment.html#cfn-codebuild-project-environment-type
	Type string `json:"Type,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCodeBuildProject_Environment) AWSCloudFormationType() string {
	return "AWS::CodeBuild::Project.Environment"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCodeBuildProject_Environment) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
