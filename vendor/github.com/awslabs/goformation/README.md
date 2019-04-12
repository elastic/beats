# AWS GoFormation

[![Build Status](https://travis-ci.org/awslabs/goformation.svg?branch=0.1.0)](https://travis-ci.org/awslabs/goformation) [![GoDoc Reference](https://godoc.org/gopkg.in/awslabs/goformation.v1?status.svg)](http://godoc.org/github.com/awslabs/goformation) ![Apache-2.0](https://img.shields.io/badge/Licence-Apache%202.0-blue.svg) 

`GoFormation` is a Go library for working with AWS CloudFormation / AWS Serverless Application Model (SAM) templates. 
- [Main features](#main-features)
- [Installation](#installation)
- [Usage](#usage)
	- [Marshalling CloudFormation/SAM described with Go structs, into YAML/JSON](#marshalling-cloudformationsam-described-with-go-structs-into-yamljson)
	- [Unmarshalling CloudFormation YAML/JSON into Go structs](#unmarshalling-cloudformation-yamljson-into-go-structs)
- [Updating CloudFormation / SAM Resources in GoFormation](#updating-cloudformation--sam-resources-in-goformation)
- [Advanced](#advanced)
	- [AWS CloudFormation Intrinsic Functions](#aws-cloudformation-intrinsic-functions)
		- [Resolving References (Ref)](#resolving-references-ref)
- [Versioning](#versioning)
- [Contributing](#contributing)

## Main features

 * Describe AWS CloudFormation and AWS SAM templates as Go objects (structs), and then turn it into JSON/YAML.
 * Parse JSON/YAML AWS CloudFormation and AWS SAM templates and turn them into Go structs.
 * Strongly typed Go structs generated for every AWS CloudFormation and AWS SAM resource.
 * Automatically generated, from the published AWS CloudFormation Resource Specification.

## Installation

As with other Go libraries, GoFormation can be installed with `go get`.

```
$ go get github.com/awslabs/goformation
```

## Usage

### Marshalling CloudFormation/SAM described with Go structs, into YAML/JSON

Below is an example of building a CloudFormation template programmatically, then outputting the resulting JSON

```go
package main

import (
	"fmt"
	"strconv"
	"time"

	"github.com/awslabs/goformation/cloudformation"
)

func main() {

	// Create a new CloudFormation template
	template := cloudformation.NewTemplate()

	// Create an Amazon SNS topic, with a unique name based off the current timestamp
	template.Resources["MyTopic"] = &cloudformation.AWSSNSTopic{
		TopicName: "my-topic-" + strconv.FormatInt(time.Now().Unix(), 10),
	}

	// Create a subscription, connected to our topic, that forwards notifications to an email address
	template.Resources["MyTopicSubscription"] = &cloudformation.AWSSNSSubscription{
		TopicArn: cloudformation.Ref("MyTopic"),
		Protocol: "email",
		Endpoint: "some.email@example.com",
	}

	// Let's see the JSON AWS CloudFormation template
	j, err := template.JSON()
	if err != nil {
		fmt.Printf("Failed to generate JSON: %s\n", err)
	} else {
		fmt.Printf("%s\n", string(j))
	}

	// and also the YAML AWS CloudFormation template
	y, err := template.YAML()
	if err != nil {
		fmt.Printf("Failed to generate YAML: %s\n", err)
	} else {
		fmt.Printf("%s\n", string(y))
	}

}
```

Would output the following JSON template:

```javascript
{
  "AWSTemplateFormatVersion": "2010-09-09",
  "Resources": {
    "MyTopic": {
      "Properties": {
        "TopicName": "my-topic-1536878058"
      },
      "Type": "AWS::SNS::Topic"
    },
    "MyTopicSubscription": {
      "Properties": {
        "Endpoint": "some.email@example.com",
        "Protocol": "email",
        "TopicArn": {
          "Ref": "MyTopic"
        }
      },
      "Type": "AWS::SNS::Subscription"
    }
  }
}
```

...and the following YAML template:

```yaml
AWSTemplateFormatVersion: 2010-09-09
Resources:
  MyTopic:
    Properties:
      TopicName: my-topic-1536878058
    Type: AWS::SNS::Topic
  MyTopicSubscription:
    Properties:
      Endpoint: some.email@example.com
      Protocol: email
      TopicArn:
        Ref: MyTopic
    Type: AWS::SNS::Subscription
```

When creating templates, you can use the following convenience functions to use [AWS CloudFormation Intrinsics](https://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html):

 - `cloudformation.Ref(logicalName string)`
 - `cloudformation.GetAtt(logicalName string, attribute string)`
 - `cloudformation.ImportValue(name string)`
 - `cloudformation.Base64(input string)`
 - `cloudformation.CIDR(ipBlock, count, cidrBits string)`
 - `cloudformation.FindInMap(mapName, topLevelKey, secondLevelKey string)`
 - `cloudformation.GetAZs(region string)`
 - `cloudformation.Join(delimiter string, values []string)`
 - `cloudformation.Select(index string, list []string)`
 - `cloudformation.Split(delimiter, source string)`
 - `cloudformation.Sub(value string)`
 - `And(conditions []string)`
 - `Equals(value1, value2 string)`
 - `If(value, ifEqual, ifNotEqual string)`
 - `Not(conditions []string)`
 - `Or(conditions []string)`

### Unmarshalling CloudFormation YAML/JSON into Go structs 

GoFormation also works the other way - parsing JSON/YAML CloudFormation/SAM templates into Go structs.

```go
package main

import (
	"log"

	"github.com/awslabs/goformation"
)

func main() {

	// Open a template from file (can be JSON or YAML)
	template, err := goformation.Open("template.yaml")
	if err != nil {
		log.Fatalf("There was an error processing the template: %s", err)
	}

	// You can extract all resources of a certain type
	// Each AWS CloudFormation resource is a strongly typed struct
	functions := template.GetAllAWSServerlessFunctionResources()
	for name, function := range functions {

		// E.g. Found a AWS::Serverless::Function named GetHelloWorld (runtime: nodejs6.10)
		log.Printf("Found a %s named %s (runtime: %s)\n", function.AWSCloudFormationType(), name, function.Runtime)

	}

	// You can also search for specific resources by their logicalId
	search := "GetHelloWorld"
	function, err := template.GetAWSServerlessFunctionWithName(search)
	if err != nil {
		log.Fatalf("Function not found")
	}

	// E.g. Found a AWS::Serverless::Function named GetHelloWorld (runtime: nodejs6.10)
	log.Printf("Found a %s named %s (runtime: %s)\n", function.AWSCloudFormationType(), search, function.Runtime)

}
```

## Updating CloudFormation / SAM Resources in GoFormation
 
AWS GoFormation contains automatically generated Go structs for every CloudFormation/SAM resource, located in the [cloudformation/](cloudformation/) directory. These can be generated, from the latest [AWS CloudFormation Resource Specification](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/cfn-resource-specification.html) published for `us-east-1` by just running `go generate`:

```
$ go generate

Generated 587 AWS CloudFormation resources from specification v1.4.2
Generated 17 AWS SAM resources from specification v2016-10-31
Generated JSON Schema: schema/cloudformation.schema.json
```

The GoFormation build pipeline automatically checks for any updated AWS CloudFormation resources on a daily basis, and creates a pull request against this repository if any are found.

## Advanced

### AWS CloudFormation Intrinsic Functions

The following [AWS CloudFormation Intrinsic Functions](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference.html) are supported in GoFormation:

- [x] [Fn::Base64](intrinsics/fnbase64.go)
- [x] [Fn::FindInMap](intrinsics/fnfindinmap.go)
- [x] [Fn::Join](intrinsics/fnjoin.go)
- [x] [Fn::Select](intrinsics/fnselect.go)
- [x] [Fn::Split](intrinsics/fnsplit.go)
- [x] [Fn::Sub](intrinsics/fnsub.go)
- [x] [Ref](intrinsics/ref.go) 
- [x] [Fn::And](intrinsics/fnand.go)      
- [x] [Fn::Equals](intrinsics/fnequals.go)  
- [x] [Fn::If](intrinsics/fnif.go)     
- [x] [Fn::Not](intrinsics/fnnot.go)      
- [x] [Fn::Or](intrinsics/fnor.go)       
- [ ] Fn::GetAtt   
- [x] [Fn::GetAZs](intrinsics/fngetazs.go)
- [ ] Fn::ImportValue

Any unsupported intrinsic functions will return `nil`.

#### Resolving References (Ref)

When converting a YAML/JSON template to go, the intrinsic 'Ref' function as implemented will resolve all of the [pseudo parameters](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-ref.html) such as `AWS::AccountId` with their default value as listed on [the bottom of this page](http://docs.aws.amazon.com/AWSCloudFormation/latest/UserGuide/intrinsic-function-reference-ref.html).

If a reference is not a pseudo parameter, GoFormation will try to resolve it within the AWS CloudFormation template. **Currently, this implementation only searches for `Parameters` with a name that matches the ref, and returns the `Default` if it has one.**


## Versioning

This library is automatically versioned and tagged using [semantic-release](https://github.com/semantic-release/semantic-release).

## Contributing

Contributions and feedback are welcome! Proposals and pull requests will be considered and responded to. For more information, see the [CONTRIBUTING](CONTRIBUTING.md) file.
