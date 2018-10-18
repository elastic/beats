package cloudformation

import (
	"encoding/json"
	"errors"
	"fmt"
)

// AWSElastiCacheSecurityGroupIngress AWS CloudFormation Resource (AWS::ElastiCache::SecurityGroupIngress)
// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-security-group-ingress.html
type AWSElastiCacheSecurityGroupIngress struct {

	// CacheSecurityGroupName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-security-group-ingress.html#cfn-elasticache-securitygroupingress-cachesecuritygroupname
	CacheSecurityGroupName string `json:"CacheSecurityGroupName,omitempty"`

	// EC2SecurityGroupName AWS CloudFormation Property
	// Required: true
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-security-group-ingress.html#cfn-elasticache-securitygroupingress-ec2securitygroupname
	EC2SecurityGroupName string `json:"EC2SecurityGroupName,omitempty"`

	// EC2SecurityGroupOwnerId AWS CloudFormation Property
	// Required: false
	// See: http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-properties-elasticache-security-group-ingress.html#cfn-elasticache-securitygroupingress-ec2securitygroupownerid
	EC2SecurityGroupOwnerId string `json:"EC2SecurityGroupOwnerId,omitempty"`

	// _deletionPolicy represents a CloudFormation DeletionPolicy
	_deletionPolicy DeletionPolicy
}

// AWSCloudFormationType returns the AWS CloudFormation resource type
func (r *AWSElastiCacheSecurityGroupIngress) AWSCloudFormationType() string {
	return "AWS::ElastiCache::SecurityGroupIngress"
}

// SetDeletionPolicy applies an AWS CloudFormation DeletionPolicy to this resource
// see: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-attribute-deletionpolicy.html
func (r *AWSElastiCacheSecurityGroupIngress) SetDeletionPolicy(policy DeletionPolicy) {
	r._deletionPolicy = policy
}

// MarshalJSON is a custom JSON marshalling hook that embeds this object into
// an AWS CloudFormation JSON resource's 'Properties' field and adds a 'Type'.
func (r AWSElastiCacheSecurityGroupIngress) MarshalJSON() ([]byte, error) {
	type Properties AWSElastiCacheSecurityGroupIngress
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
func (r *AWSElastiCacheSecurityGroupIngress) UnmarshalJSON(b []byte) error {
	type Properties AWSElastiCacheSecurityGroupIngress
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
		*r = AWSElastiCacheSecurityGroupIngress(*res.Properties)
	}

	return nil
}

// GetAllAWSElastiCacheSecurityGroupIngressResources retrieves all AWSElastiCacheSecurityGroupIngress items from an AWS CloudFormation template
func (t *Template) GetAllAWSElastiCacheSecurityGroupIngressResources() map[string]AWSElastiCacheSecurityGroupIngress {
	results := map[string]AWSElastiCacheSecurityGroupIngress{}
	for name, untyped := range t.Resources {
		switch resource := untyped.(type) {
		case AWSElastiCacheSecurityGroupIngress:
			// We found a strongly typed resource of the correct type; use it
			results[name] = resource
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElastiCache::SecurityGroupIngress" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElastiCacheSecurityGroupIngress
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

// GetAWSElastiCacheSecurityGroupIngressWithName retrieves all AWSElastiCacheSecurityGroupIngress items from an AWS CloudFormation template
// whose logical ID matches the provided name. Returns an error if not found.
func (t *Template) GetAWSElastiCacheSecurityGroupIngressWithName(name string) (AWSElastiCacheSecurityGroupIngress, error) {
	if untyped, ok := t.Resources[name]; ok {
		switch resource := untyped.(type) {
		case AWSElastiCacheSecurityGroupIngress:
			// We found a strongly typed resource of the correct type; use it
			return resource, nil
		case map[string]interface{}:
			// We found an untyped resource (likely from JSON) which *might* be
			// the correct type, but we need to check it's 'Type' field
			if resType, ok := resource["Type"]; ok {
				if resType == "AWS::ElastiCache::SecurityGroupIngress" {
					// The resource is correct, unmarshal it into the results
					if b, err := json.Marshal(resource); err == nil {
						var result AWSElastiCacheSecurityGroupIngress
						if err := json.Unmarshal(b, &result); err == nil {
							return result, nil
						}
					}
				}
			}
		}
	}
	return AWSElastiCacheSecurityGroupIngress{}, errors.New("resource not found")
}
