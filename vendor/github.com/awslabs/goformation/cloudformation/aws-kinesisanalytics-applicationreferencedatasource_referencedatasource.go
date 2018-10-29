package cloudformation

// AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceDataSource AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource.ReferenceDataSource)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referencedatasource.html
type AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceDataSource struct {

	// ReferenceSchema AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referencedatasource.html#cfn-kinesisanalytics-applicationreferencedatasource-referencedatasource-referenceschema
	ReferenceSchema *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceSchema `json:"ReferenceSchema,omitempty"`

	// S3ReferenceDataSource AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referencedatasource.html#cfn-kinesisanalytics-applicationreferencedatasource-referencedatasource-s3referencedatasource
	S3ReferenceDataSource *AWSKinesisAnalyticsApplicationReferenceDataSource_S3ReferenceDataSource `json:"S3ReferenceDataSource,omitempty"`

	// TableName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referencedatasource.html#cfn-kinesisanalytics-applicationreferencedatasource-referencedatasource-tablename
	TableName string `json:"TableName,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceDataSource) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource.ReferenceDataSource"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceDataSource) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
