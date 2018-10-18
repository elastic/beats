package cloudformation

// AWSKinesisAnalyticsApplicationOutput_KinesisFirehoseOutput AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationOutput.KinesisFirehoseOutput)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-kinesisfirehoseoutput.html
type AWSKinesisAnalyticsApplicationOutput_KinesisFirehoseOutput struct {

	// ResourceARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-kinesisfirehoseoutput.html#cfn-kinesisanalytics-applicationoutput-kinesisfirehoseoutput-resourcearn
	ResourceARN string `json:"ResourceARN,omitempty"`

	// RoleARN AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-kinesisfirehoseoutput.html#cfn-kinesisanalytics-applicationoutput-kinesisfirehoseoutput-rolearn
	RoleARN string `json:"RoleARN,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationOutput_KinesisFirehoseOutput) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationOutput.KinesisFirehoseOutput"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationOutput_KinesisFirehoseOutput) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
