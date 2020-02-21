package elasticsearch

import (
	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// Domain_ZoneAwarenessConfig AWS CloudFormation Resource (AWS::Elasticsearch::Domain.ZoneAwarenessConfig)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-zoneawarenessconfig.html
type Domain_ZoneAwarenessConfig struct {

	// AvailabilityZoneCount AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticsearch-domain-zoneawarenessconfig.html#cfn-elasticsearch-domain-zoneawarenessconfig-availabilityzonecount
	AvailabilityZoneCount int `json:"AvailabilityZoneCount,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *Domain_ZoneAwarenessConfig) AWSCloudFormationType() string {
	return "AWS::Elasticsearch::Domain.ZoneAwarenessConfig"
}
