package cloudformation

// AWSElasticsearchDomain_SnapshotOptions AWS CloudFormation Resource (AWS::Elasticsearch::Domain.SnapshotOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-snapshotoptions.html
type AWSElasticsearchDomain_SnapshotOptions struct {

	// AutomatedSnapshotStartHour AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-snapshotoptions.html#cfn-elasticsearch-domain-snapshotoptions-automatedsnapshotstarthour
	AutomatedSnapshotStartHour int `json:"AutomatedSnapshotStartHour,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElasticsearchDomain_SnapshotOptions) AWSCloudFormationType() string {
	return "AWS::Elasticsearch::Domain.SnapshotOptions"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElasticsearchDomain_SnapshotOptions) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}
