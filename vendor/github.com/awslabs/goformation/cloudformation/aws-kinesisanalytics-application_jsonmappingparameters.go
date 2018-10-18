package cloudformation

// AWSKinesisAnalyticsApplication_JSONMappingParameters AWS CloudFormation Resource (AWS::KinesisAnalytics::Application.JSONMappingParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-jsonmappingparameters.html
type AWSKinesisAnalyticsApplication_JSONMappingParameters struct {

	// RecordRowPath AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-application-jsonmappingparameters.html#cfn-kinesisanalytics-application-jsonmappingparameters-recordrowpath
	RecordRowPath string `json:"RecordRowPath,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplication_JSONMappingParameters) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::Application.JSONMappingParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplication_JSONMappingParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
