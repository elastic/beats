package cloudformation

// AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceSchema AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationReferenceDataSource.ReferenceSchema)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referenceschema.html
type AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceSchema struct {

	// RecordColumns AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referenceschema.html#cfn-kinesisanalytics-applicationreferencedatasource-referenceschema-recordcolumns
	RecordColumns []AWSKinesisAnalyticsApplicationReferenceDataSource_RecordColumn `json:"RecordColumns,omitempty"`

	// RecordEncoding AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referenceschema.html#cfn-kinesisanalytics-applicationreferencedatasource-referenceschema-recordencoding
	RecordEncoding string `json:"RecordEncoding,omitempty"`

	// RecordFormat AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationreferencedatasource-referenceschema.html#cfn-kinesisanalytics-applicationreferencedatasource-referenceschema-recordformat
	RecordFormat *AWSKinesisAnalyticsApplicationReferenceDataSource_RecordFormat `json:"RecordFormat,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceSchema) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationReferenceDataSource.ReferenceSchema"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationReferenceDataSource_ReferenceSchema) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
