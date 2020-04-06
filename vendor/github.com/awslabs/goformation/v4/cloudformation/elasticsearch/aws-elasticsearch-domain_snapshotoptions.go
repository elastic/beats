package elasticsearch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Domain_SnapshotOptions AWS CloudFormation Resource (AWS::Elasticsearch::Domain.SnapshotOptions)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-snapshotoptions.html
type Domain_SnapshotOptions struct {

	// AutomatedSnapshotStartHour AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-snapshotoptions.html#cfn-elasticsearch-domain-snapshotoptions-automatedsnapshotstarthour
	AutomatedSnapshotStartHour int `json:"AutomatedSnapshotStartHour,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Domain_SnapshotOptions) AWSCloudFormationType() string {
	return "AWS::Elasticsearch::Domain.SnapshotOptions"
}
