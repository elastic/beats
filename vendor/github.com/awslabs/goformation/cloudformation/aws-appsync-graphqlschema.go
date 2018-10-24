package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSAppSyncGraphQLSchema AWS CloudFormation Resource (AWS::AppSync::GraphQLSchema)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-appsync-graphqlschema.html
type AWSAppSyncGraphQLSchema struct {

	// ApiId AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-appsync-graphqlschema.html#cfn-appsync-graphqlschema-apiid
	ApiId string `json:"ApiId,omitempty"`

	// Definition AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-appsync-graphqlschema.html#cfn-appsync-graphqlschema-definition
	Definition string `json:"Definition,omitempty"`

	// DefinitionS3Location AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-appsync-graphqlschema.html#cfn-appsync-graphqlschema-definitions3location
	DefinitionS3Location string `json:"DefinitionS3Location,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSAppSyncGraphQLSchema) AWSCloudFormationType() string {
	return "AWS::AppSync::GraphQLSchema"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSAppSyncGraphQLSchema) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSAppSyncGraphQLSchema) MarshalJSON() ([]byte, error) {
	type Properties AWSAppSyncGraphQLSchema
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
func (r *AWSAppSyncGraphQLSchema) UnmarshalJSON(b []byte) error {
	type Properties AWSAppSyncGraphQLSchema
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
		*r = AWSAppSyncGraphQLSchema(*res.Properties)
	}

	return nil
}

// GetAllAWSAppSyncGraphQLSchemaResources retrieves all AWSAppSyncGraphQLSchema items from an AWS CloudFormation template
func (t *Template) GetAllAWSAppSyncGraphQLSchemaResources() map[string]AWSAppSyncGraphQLSchema {
	results := map[string]AWSAppSyncGraphQLSchema{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSAppSyncGraphQLSchema:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::AppSync::GraphQLSchema" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSAppSyncGraphQLSchema
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

// GetAWSAppSyncGraphQLSchemaWithName retrieves all AWSAppSyncGraphQLSchema items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSAppSyncGraphQLSchemaWithName(name string) (AWSAppSyncGraphQLSchema, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSAppSyncGraphQLSchema:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::AppSync::GraphQLSchema" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSAppSyncGraphQLSchema
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSAppSyncGraphQLSchema{}, errors.New("resource not found")
}
