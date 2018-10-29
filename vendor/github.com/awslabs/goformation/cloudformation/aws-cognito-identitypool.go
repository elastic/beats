package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSCognitoIdentityPool AWS CloudFormation Resource (AWS::Cognito::IdentityPool)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html
type AWSCognitoIdentityPool struct {

	// AllowUnauthenticatedIdentities AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-allowunauthenticatedidentities
	AllowUnauthenticatedIdentities bool `json:"AllowUnauthenticatedIdentities,omitempty"`

	// CognitoEvents AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-cognitoevents
	CognitoEvents interface{} `json:"CognitoEvents,omitempty"`

	// CognitoIdentityProviders AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-cognitoidentityproviders
	CognitoIdentityProviders []AWSCognitoIdentityPool_CognitoIdentityProvider `json:"CognitoIdentityProviders,omitempty"`

	// CognitoStreams AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-cognitostreams
	CognitoStreams *AWSCognitoIdentityPool_CognitoStreams `json:"CognitoStreams,omitempty"`

	// DeveloperProviderName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-developerprovidername
	DeveloperProviderName string `json:"DeveloperProviderName,omitempty"`

	// IdentityPoolName AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-identitypoolname
	IdentityPoolName string `json:"IdentityPoolName,omitempty"`

	// OpenIdConnectProviderARNs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-openidconnectproviderarns
	OpenIdConnectProviderARNs []string `json:"OpenIdConnectProviderARNs,omitempty"`

	// PushSync AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-pushsync
	PushSync *AWSCognitoIdentityPool_PushSync `json:"PushSync,omitempty"`

	// SamlProviderARNs AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-samlproviderarns
	SamlProviderARNs []string `json:"SamlProviderARNs,omitempty"`

	// SupportedLoginProviders AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-cognito-identitypool.html#cfn-cognito-identitypool-supportedloginproviders
	SupportedLoginProviders interface{} `json:"SupportedLoginProviders,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSCognitoIdentityPool) AWSCloudFormationType() string {
	return "AWS::Cognito::IdentityPool"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSCognitoIdentityPool) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSCognitoIdentityPool) MarshalJSON() ([]byte, error) {
	type Properties AWSCognitoIdentityPool
	return json.Marshal(&struct {
		Type           string
		Properties     Properties
		DeletionPolicy DeletionPolicy `json:"DeletionPolicy,omitempty"`
	}{
		Type:           r.AWSCloudFormationType(),
		Properties:     (Properties)(r),
		DeletionPolicy: r._deletionPolicy,
	})
}

// UnmarshalJSON is a custom JSON unmarshalling hook that strips the outer
// AWS CloudFormation resource object, and just keeps the 'Properties' field.
func (r *AWSCognitoIdentityPool) UnmarshalJSON(b []byte) error {
	type Properties AWSCognitoIdentityPool
	res := &struct {
		Type       string
		Properties *Properties
	}{}
	if err := json.Unmarshal(b, &res); err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return err
	}

	// If the resource has no Properties set, it could be nil
	if res.Properties != nil {
		*r = AWSCognitoIdentityPool(*res.Properties)
	}

	return nil
}

// GetAllAWSCognitoIdentityPoolResources retrieves all AWSCognitoIdentityPool items from an AWS CloudFormation template
func (t *Template) GetAllAWSCognitoIdentityPoolResources() map[string]AWSCognitoIdentityPool {
	results := map[string]AWSCognitoIdentityPool{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSCognitoIdentityPool:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Cognito::IdentityPool" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCognitoIdentityPool
						if err := json.Unmarshal(b, &result); err == nil {
							results[name] = result
						}
					}
				}
			}
		}
	}
	return results
}

// GetAWSCognitoIdentityPoolWithName retrieves all AWSCognitoIdentityPool items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSCognitoIdentityPoolWithName(name string) (AWSCognitoIdentityPool, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSCognitoIdentityPool:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::Cognito::IdentityPool" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSCognitoIdentityPool
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSCognitoIdentityPool{}, errors.New("resource not found")
}
