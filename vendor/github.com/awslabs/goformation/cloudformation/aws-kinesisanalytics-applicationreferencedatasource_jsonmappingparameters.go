package cloudformation

// AWSKinesisAnalyticsApplicationReferenceDataSource_JSONMappingParameters AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource.JSONMappingParameters)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-jsonmappingparameters.html
type AWSKinesisAnalyticsApplicationReferenceDataSource_JSONMappingParameters struct {

	// RecordRowPath AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-jsonmappingparameters.html#cfn-kinesisanalytics-applicationreferencedatasource-jsonmappingparameters-recordrowpath
	RecordRowPath string `json:"RecordRowPath,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_JSONMappingParameters) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource.JSONMappingParameters"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_JSONMappingParameters) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
