package s3

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// BucketPolicy AWS CloudFormation Resource (AWS::S3::BucketPolicy)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-policy.html
type BucketPolicy struct {

	// Bucket AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-policy.html#aws-properties-s3-policy-bucket
	Bucket string `json:"Bucket,omitempty"`

	// PolicyDocument AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-s3-policy.html#aws-properties-s3-policy-policydocument
	PolicyDocument interface{} `json:"PolicyDocument,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *BucketPolicy) AWSCloudFormationType() string {
	return "AWS::S3::BucketPolicy"
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r BucketPolicy) MarshalJSON() ([]byte, error) {
	type Properties BucketPolicy
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DependsOn      []string                `json:"DependsOn,omitempty"`
		Metadata       map[string]interface{}  `json:"Metadata,omitempty"`
		DeletionPolicy policies.DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DependsOn:      r.AWSCloudFormationDependsOn,
		Metadata:       r.AWSCloudFormationMetadata,
		DeletionPolicy: r.AWSCloudFormationDeletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *BucketPolicy) UnmarshalJSON(b []byte) error {
	type Properties BucketPolicy
	res := &struct {
		Type           string
		Properties     *Properties
		DependsOn      []string
		Metadata       map[string]interface{}
		DeletionPolicy string
	}{}

	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields() // Force error if unknown field is found

	if err := dec.Decode(&res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = BucketPolicy(*res.Properties)
	}
	if res.DependsOn != nil {
		r.AWSCloudFormationDependsOn = res.DependsOn
	}
	if res.Metadata != nil {
		r.AWSCloudFormationMetadata = res.Metadata
	}
	if res.DeletionPolicy != "" {
		r.AWSCloudFormationDeletionPolicy = policies.DeletionPolicy(res.DeletionPolicy)
	}
	return nil
}
