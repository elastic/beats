package pinpoint

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/awslabs/goformation/v4/cloudformation/policies"
)

// APNSSandboxChannel AWS CloudFormation Resource (AWS::Pinpoint::APNSSandboxChannel)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html
type APNSSandboxChannel struct {

	// ApplicationId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-applicationid
	ApplicationId string `json:"ApplicationId,omitempty"`

	// BundleId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-bundleid
	BundleId string `json:"BundleId,omitempty"`

	// Certificate AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-certificate
	Certificate string `json:"Certificate,omitempty"`

	// DefaultAuthenticationMethod AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-defaultauthenticationmethod
	DefaultAuthenticationMethod string `json:"DefaultAuthenticationMethod,omitempty"`

	// Enabled AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-enabled
	Enabled bool `json:"Enabled,omitempty"`

	// PrivateKey AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-privatekey
	PrivateKey string `json:"PrivateKey,omitempty"`

	// TeamId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-teamid
	TeamId string `json:"TeamId,omitempty"`

	// TokenKey AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-tokenkey
	TokenKey string `json:"TokenKey,omitempty"`

	// TokenKeyId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-pinpoint-apnssandboxchannel.html#cfn-pinpoint-apnssandboxchannel-tokenkeyid
	TokenKeyId string `json:"TokenKeyId,omitempty"`

	// AWSCloudFormationDeletionPolicy represents a CloudFormation DeletionPolicy
	AWSCloudFormationDeletionPolicy policies.DeletionPolicy `json:"-"`

	// AWSCloudFormationDependsOn stores the logical ID of the resources to be created before this resource
	AWSCloudFormationDependsOn []string `json:"-"`

	// AWSCloudFormationMetadata stores structured data associated with this resource
	AWSCloudFormationMetadata map[string]interface{} `json:"-"`
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *APNSSandboxChannel) AWSCloudFormationType() string {
	return "AWS::Pinpoint::APNSSandboxChannel"
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r APNSSandboxChannel) MarshalJSON() ([]byte, error) {
	type Properties APNSSandboxChannel
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
func (r *APNSSandboxChannel) UnmarshalJSON(b []byte) error {
	type Properties APNSSandboxChannel
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
		*r = APNSSandboxChannel(*res.Properties)
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
