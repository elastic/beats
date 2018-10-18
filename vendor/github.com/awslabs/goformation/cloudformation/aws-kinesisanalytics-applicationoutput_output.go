package cloudformation

// AWSKinesisAnalyticsApplicationOutput_Output AWS CloudFormation Resource (AWS::KinesisAnalytics::ApplicationOutput.Output)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html
type AWSKinesisAnalyticsApplicationOutput_Output struct {

	// DestinationSchema AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html#cfn-kinesisanalytics-applicationoutput-output-destinationschema
	DestinationSchema *AWSKinesisAnalyticsApplicationOutput_DestinationSchema `json:"DestinationSchema,omitempty"`

	// KinesisFirehoseOutput AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html#cfn-kinesisanalytics-applicationoutput-output-kinesisfirehoseoutput
	KinesisFirehoseOutput *AWSKinesisAnalyticsApplicationOutput_KinesisFirehoseOutput `json:"KinesisFirehoseOutput,omitempty"`

	// KinesisStreamsOutput AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html#cfn-kinesisanalytics-applicationoutput-output-kinesisstreamsoutput
	KinesisStreamsOutput *AWSKinesisAnalyticsApplicationOutput_KinesisStreamsOutput `json:"KinesisStreamsOutput,omitempty"`

	// LambdaOutput AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html#cfn-kinesisanalytics-applicationoutput-output-lambdaoutput
	LambdaOutput *AWSKinesisAnalyticsApplicationOutput_LambdaOutput `json:"LambdaOutput,omitempty"`

	// Name AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-kinesisanalytics-applicationoutput-output.html#cfn-kinesisanalytics-applicationoutput-output-name
	Name string `json:"Name,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSKinesisAnalyticsApplicationOutput_Output) AWSCloudFormationType() string {
	return "AWS::KinesisAnalytics::ApplicationOutput.Output"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSKinesisAnalyticsApplicationOutput_Output) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
