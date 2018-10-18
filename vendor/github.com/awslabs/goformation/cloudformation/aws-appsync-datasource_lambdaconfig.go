package cloudformation

// AWSAppSyncDataSource_LambdaConfig AWS CloudFormation Resource (AWS::AppSync::DataSource.LambdaConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-lambdaconfig.html
type AWSAppSyncDataSource_LambdaConfig struct {

	// LambdaFunctionArn AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-appsync-datasource-lambdaconfig.html#cfn-appsync-datasource-lambdaconfig-lambdafunctionarn
	LambdaFunctionArn string `json:"LambdaFunctionArn,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAppSyncDataSource_LambdaConfig) AWSCloudFormationType() string {
	return "AWS::AppSync::DataSource.LambdaConfig"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAppSyncDataSource_LambdaConfig) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
