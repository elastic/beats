// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/awslabs/goformation/cloudformation"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/x-pack/functionbeat/function/core"
	"github.com/elastic/beats/x-pack/functionbeat/function/provider"
)

// zipData stores the data on the zip to be deployed
type zipData struct {
	content  []byte
	checksum string
}

// templateData stores the template and its metadata required to deploy it
type templateData struct {
	json     []byte
	checksum string
	key      string
	url      string
	codeKey  string
	zip      zipData
}

type defaultTemplateBuilder struct {
	provider provider.Provider
	log      *logp.Logger
	endpoint string
	bucket   string
}

const (
	keyPrefix = "functionbeat-deployment/"
)

func NewTemplateBuilder(log *logp.Logger, cfg *common.Config, p provider.Provider) (provider.TemplateBuilder, error) {
	config := &Config{}
	if err := cfg.Unpack(config); err != nil {
		return nil, err
	}

	return &defaultTemplateBuilder{
		provider: p,
		log:      log,
		endpoint: config.Endpoint,
		bucket:   string(config.DeployBucket),
	}, nil
}

func (d *defaultTemplateBuilder) findFunction(name string) (installer, error) {
	fn, err := d.provider.FindFunctionByName(name)
	if err != nil {
		return nil, err
	}

	function, ok := fn.(installer)
	if !ok {
		return nil, errors.New("incompatible type received, expecting: 'functionManager'")
	}

	return function, nil
}

// execute generates a template
func (d *defaultTemplateBuilder) execute(name string) (templateData, error) {
	d.log.Debug("Compressing all assets into an artifact")
	content, err := core.MakeZip("aws")
	if err != nil {
		return templateData{}, err
	}
	d.log.Debugf("Compression is successful (zip size: %d bytes)", len(content))

	function, err := d.findFunction(name)
	if err != nil {
		return templateData{}, err
	}

	fnTemplate := function.Template()

	zipChecksum := checksum(content)
	codeKey := keyPrefix + name + "/" + zipChecksum + "/functionbeat.zip"
	to := d.template(function, name, codeKey)
	if err := mergeTemplate(to, fnTemplate); err != nil {
		return templateData{}, err
	}

	templateJSON, err := to.JSON()
	if err != nil {
		return templateData{}, err
	}

	templateChecksum := checksum(templateJSON)
	templateKey := keyPrefix + name + "/" + templateChecksum + "/cloudformation-template-create.json"
	templateURL := "https://" + d.bucket + "." + d.endpoint + "/" + templateKey

	return templateData{
		json:     templateJSON,
		checksum: templateChecksum,
		key:      templateKey,
		url:      templateURL,
		codeKey:  codeKey,
		zip: zipData{
			checksum: zipChecksum,
			content:  content,
		},
	}, nil
}

func (d *defaultTemplateBuilder) template(function installer, name, codeLoc string) *cloudformation.Template {
	lambdaConfig := function.LambdaConfig()

	prefix := func(s string) string {
		return normalizeResourceName("fnb" + name + s)
	}

	// AWS variables references:.
	// AWS::Partition: aws, aws-cn, aws-gov.
	// AWS::Region: us-east-1, us-east-2, ap-northeast-3,
	// AWS::AccountId: account id for the current request.
	// AWS::URLSuffix: amazonaws.com
	//
	// Documentation: https://docs.aws.amazon.com/AWSCloudFormation/latest/APIReference/Welcome.html
	// Intrinsic function reference: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html

	template := cloudformation.NewTemplate()

	role := lambdaConfig.Role
	dependsOn := make([]string, 0)
	if lambdaConfig.Role == "" {
		d.log.Infof("No role is configured for function %s, creating a custom role.", name)

		roleRes := prefix("") + "IAMRoleLambdaExecution"
		template.Resources[roleRes] = d.roleTemplate(function, name)
		role = cloudformation.GetAtt(roleRes, "Arn")
		dependsOn = []string{roleRes}
	}

	// Configure the Dead letter, any failed events will be send to the configured amazon resource name.
	var dlc *cloudformation.AWSLambdaFunction_DeadLetterConfig
	if lambdaConfig.DeadLetterConfig != nil && len(lambdaConfig.DeadLetterConfig.TargetArn) != 0 {
		dlc = &cloudformation.AWSLambdaFunction_DeadLetterConfig{
			TargetArn: lambdaConfig.DeadLetterConfig.TargetArn,
		}
	}

	// Configure VPC
	var vcpConf *cloudformation.AWSLambdaFunction_VpcConfig
	if lambdaConfig.VPCConfig != nil && len(lambdaConfig.VPCConfig.SecurityGroupIDs) != 0 && len(lambdaConfig.VPCConfig.SubnetIDs) != 0 {
		vcpConf = &cloudformation.AWSLambdaFunction_VpcConfig{
			SecurityGroupIds: lambdaConfig.VPCConfig.SecurityGroupIDs,
			SubnetIds:        lambdaConfig.VPCConfig.SubnetIDs,
		}
	}

	// Create the lambda
	// Doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-lambda-function.html
	template.Resources[prefix("")] = &AWSLambdaFunction{
		AWSLambdaFunction: &cloudformation.AWSLambdaFunction{
			Code: &cloudformation.AWSLambdaFunction_Code{
				S3Bucket: d.bucket,
				S3Key:    codeLoc,
			},
			Description: lambdaConfig.Description,
			Environment: &cloudformation.AWSLambdaFunction_Environment{
				// Configure which function need to be run by the lambda function.
				Variables: map[string]string{
					"BEAT_STRICT_PERMS": "false", // Disable any check on disk, we are running with really differents permission on lambda.
					"ENABLED_FUNCTIONS": name,
				},
			},
			DeadLetterConfig:             dlc,
			VpcConfig:                    vcpConf,
			FunctionName:                 name,
			Role:                         role,
			Runtime:                      runtime,
			Handler:                      handlerName,
			MemorySize:                   lambdaConfig.MemorySize.Megabytes(),
			ReservedConcurrentExecutions: lambdaConfig.Concurrency,
			Timeout:                      int(lambdaConfig.Timeout.Seconds()),
		},
		DependsOn: dependsOn,
	}

	// Create the log group for the specific function lambda.
	template.Resources[prefix("LogGroup")] = &cloudformation.AWSLogsLogGroup{
		LogGroupName: "/aws/lambda/" + name,
	}

	return template
}

func (d *defaultTemplateBuilder) roleTemplate(function installer, name string) *cloudformation.AWSIAMRole {
	// Default policies to writes logs from the Lambda.
	policies := []cloudformation.AWSIAMRole_Policy{
		cloudformation.AWSIAMRole_Policy{
			PolicyName: cloudformation.Join("-", []string{"fnb", "lambda", name}),
			PolicyDocument: map[string]interface{}{
				"Statement": []map[string]interface{}{
					map[string]interface{}{
						"Action": []string{"logs:CreateLogStream", "logs:PutLogEvents"},
						"Effect": "Allow",
						"Resource": []string{
							cloudformation.Sub("arn:${AWS::Partition}:logs:${AWS::Region}:${AWS::AccountId}:log-group:/aws/lambda/" + name + ":*"),
						},
					},
				},
			},
		},
	}

	// Merge any specific policies from the service.
	policies = append(policies, function.Policies()...)

	// Create the roles for the lambda.
	// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-role.html
	return &cloudformation.AWSIAMRole{
		AssumeRolePolicyDocument: map[string]interface{}{
			"Statement": []interface{}{
				map[string]interface{}{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Principal": map[string]interface{}{
						"Service": cloudformation.Join("", []string{
							"lambda.",
							cloudformation.Ref("AWS::URLSuffix"),
						}),
					},
				},
			},
		},
		Path:     "/",
		RoleName: "functionbeat-lambda-" + name,
		// Allow the lambda to write log to cloudwatch logs.
		// doc: https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/aws-resource-iam-policy.html
		Policies: policies,
	}
}

// RawTemplate generates a template and returns it in a string
func (d *defaultTemplateBuilder) RawTemplate(name string) (string, error) {
	data, err := d.execute(name)
	return string(data.json), err
}

// mergeTemplate takes two cloudformation and merge them, if a key already exist we return an error.
func mergeTemplate(to, from *cloudformation.Template) error {
	merge := func(m1 map[string]interface{}, m2 map[string]interface{}) error {
		for k, v := range m2 {
			if _, ok := m1[k]; ok {
				return fmt.Errorf("key %s already exist in the template map", k)
			}
			m1[k] = v
		}
		return nil
	}

	err := merge(to.Parameters, from.Parameters)
	if err != nil {
		return err
	}

	err = merge(to.Mappings, from.Mappings)
	if err != nil {
		return err
	}

	err = merge(to.Conditions, from.Conditions)
	if err != nil {
		return err
	}

	err = merge(to.Resources, from.Resources)
	if err != nil {
		return err
	}

	err = merge(to.Outputs, from.Outputs)
	if err != nil {
		return err
	}

	return nil
}

func checksum(data []byte) string {
	sha := sha256.Sum256(data)
	return base64.RawURLEncoding.EncodeToString(sha[:])
}
