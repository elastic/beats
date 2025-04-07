---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/filebeat-module-awsfargate.html
---

# AWS Fargate module [filebeat-module-awsfargate]

::::{warning}
This functionality is in beta and is subject to change. The design and code is less mature than official GA features and is being provided as-is with no warranties. Beta features are not subject to the support SLA of official GA features.
::::


This module can be used to collect container logs from Amazon ECS on Fargate. It uses filebeat `awscloudwatch` input to get log files from one or more log streams in AWS CloudWatch. Logs from all containers in Fargate launch type tasks can be sent to CloudWatch by adding the `awslogs` log driver under `logConfiguration` section in the task definition. For example, `logConfiguration` can be added into the task definition by adding this section into the `containerDefinitions`:

```json
{
   "logDriver":"awslogs",
   "options":{
      "awslogs-group":"awslogs-wordpress",
      "awslogs-region":"us-west-2",
      "awslogs-stream-prefix":"awslogs-example"
   }
}
```

The `awsfargate` module requires AWS credentials configuration in order to make AWS API calls. Users can either use `access_key_id`, `secret_access_key` and/or `session_token`, or use `role_arn` AWS IAM role, or use shared AWS credentials file.

Please see [AWS credentials options](#awsfargate-credentials) for more details.


## Module configuration [_module_configuration_2]

Example config:

```yaml
- module: awsfargate
  log:
    enabled: true
    var.credential_profile_name: test-filebeat
    var.log_group_arn: arn:aws:logs:us-east-1:1234567890:log-group:/ecs/test-log-group:*
```

**`var.log_group_arn`**
:   ARN of the log group to collect logs from.

**`var.log_group_name`**
:   Name of the log group to collect logs from. Note: region_name is required when log_group_name is given.

**`var.region_name`**
:   Region that the specified log group belongs to.

**`var.log_streams`**
:   A list of strings of log streams names that Filebeat collect log events from.

**`var.log_stream_prefix`**
:   A string to filter the results to include only log events from log streams that have names starting with this prefix.

**`var.start_position`**
:   `start_position` allows user to specify if this input should read log files from the `beginning` or from the `end`.

    * `beginning`: reads from the beginning of the log group (default).
    * `end`: read only new messages from current time minus `scan_frequency` going forward


**`var.scan_frequency`**
:   This config parameter sets how often Filebeat checks for new log events from the specified log group. Default `scan_frequency` is 1 minute, which means Filebeat will sleep for 1 minute before querying for new logs again.

**`var.api_timeout`**
:   The maximum duration of AWS API can take. If it exceeds the timeout, AWS API will be interrupted. The default AWS API timeout for a message is 120 seconds. The minimum is 0 seconds.

**`var.api_sleep`**
:   This is used to sleep between AWS `FilterLogEvents` API calls inside the same collection period. `FilterLogEvents` API has a quota of 5 transactions per second (TPS)/account/Region. By default, `api_sleep` is 200 ms. This value should only be adjusted when there are multiple Filebeats or multiple Filebeat inputs collecting logs from the same region and AWS account.

**`var.shared_credential_file`**
:   Filename of AWS credential file.

**`var.credential_profile_name`**
:   AWS credential profile name.

**`var.access_key_id`**
:   First part of access key.

**`var.secret_access_key`**
:   Second part of access key.

**`var.session_token`**
:   Required when using temporary security credentials.

**`var.role_arn`**
:   AWS IAM Role to assume.

**`var.endpoint`**
:   The custom endpoint used to access AWS APIs.


## AWS Credentials Configuration [awsfargate-credentials]

To configure AWS credentials, either put the credentials into the Filebeat configuration, or use a shared credentials file, as shown in the following examples.


### Configuration parameters [_configuration_parameters_3]

* **access_key_id**: first part of access key.
* **secret_access_key**: second part of access key.
* **session_token**: required when using temporary security credentials.
* **credential_profile_name**: profile name in shared credentials file.
* **shared_credential_file**: directory of the shared credentials file.
* **role_arn**: AWS IAM Role to assume.
* **external_id**: external ID to use when assuming a role in another account, see [the AWS documentation for use of external IDs](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create_for-user_externalid.html).
* **proxy_url**: URL of the proxy to use to connect to AWS web services. The syntax is `http(s)://<IP/Hostname>:<port>`
* **fips_enabled**: Enabling this option instructs Filebeat to use the FIPS endpoint of a service. All services used by Filebeat are FIPS compatible except for `tagging` but only certain regions are FIPS compatible. See [https://aws.amazon.com/compliance/fips/](https://aws.amazon.com/compliance/fips/) or the appropriate service page, [https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html](https://docs.aws.amazon.com/general/latest/gr/aws-service-information.html), for a full list of FIPS endpoints and regions.
* **ssl**: This specifies SSL/TLS configuration. If the ssl section is missing, the host’s CAs are used for HTTPS connections. See [SSL](/reference/filebeat/configuration-ssl.md) for more information.
* **default_region**: Default region to query if no other region is set. Most AWS services offer a regional endpoint that can be used to make requests. Some services, such as IAM, do not support regions. If a region is not provided by any other way (environment variable, credential or instance profile), the value set here will be used.
* **assume_role.duration**: The duration of the requested assume role session. Defaults to 15m when not set. AWS allows a maximum session duration between 1h and 12h depending on your maximum session duration policies.
* **assume_role.expiry_window**: The expiry_window will allow refreshing the session prior to its expiration. This is beneficial to prevent expiring tokens from causing requests to fail with an ExpiredTokenException.


### Supported Formats [_supported_formats_3]

::::{note}
The examples in this section refer to Metricbeat, but the credential options for authentication with AWS are the same no matter which Beat is being used.
::::


* Use `access_key_id`, `secret_access_key`, and/or `session_token`

Users can either put the credentials into the Metricbeat module configuration or use environment variable `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` and/or `AWS_SESSION_TOKEN` instead.

If running on Docker, these environment variables should be added as a part of the docker command. For example, with Metricbeat:

```bash
$ docker run -e AWS_ACCESS_KEY_ID=abcd -e AWS_SECRET_ACCESS_KEY=abcd -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  access_key_id: ${AWS_ACCESS_KEY_ID}
  secret_access_key: ${AWS_SECRET_ACCESS_KEY}
  session_token: ${AWS_SESSION_TOKEN}
  metricsets:
    - ec2
```

Environment variables can also be added through a file. For example:

```bash
$ cat env.list
AWS_ACCESS_KEY_ID=abcd
AWS_SECRET_ACCESS_KEY=abcd

$ docker run --env-file env.list -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

* Use `credential_profile_name` and/or `shared_credential_file`

If `access_key_id`, `secret_access_key` and `role_arn` are all not given, then filebeat will check for `credential_profile_name`. If you use different credentials for different tools or applications, you can use profiles to configure multiple access keys in the same configuration file. If there is no `credential_profile_name` given, the default profile will be used.

`shared_credential_file` is optional to specify the directory of your shared credentials file. If it’s empty, the default directory will be used. In Windows, shared credentials file is at `C:\Users\<yourUserName>\.aws\credentials`. For Linux, macOS or Unix, the file is located at `~/.aws/credentials`. When running as a service, the home path depends on the user that manages the service, so the `shared_credential_file` parameter can be used to avoid ambiguity. Please see [Create Shared Credentials File](https://docs.aws.amazon.com/ses/latest/DeveloperGuide/create-shared-credentials-file.md) for more details.

* Use `role_arn`

`role_arn` is used to specify which AWS IAM role to assume for generating temporary credentials. If `role_arn` is given, filebeat will check if access keys are given. If not, filebeat will check for credential profile name. If neither is given, default credential profile will be used. Please make sure credentials are given under either a credential profile or access keys.

If running on Docker, the credential file needs to be provided via a volume mount. For example, with Metricbeat:

```bash
docker run -d --name=metricbeat --user=root --volume="$(pwd)/metricbeat.aws.yml:/usr/share/metricbeat/metricbeat.yml:ro" --volume="/Users/foo/.aws/credentials:/usr/share/metricbeat/credentials:ro" docker.elastic.co/beats/metricbeat:7.11.1 metricbeat -e -E cloud.auth=elastic:1234 -E cloud.id=test-aws:1234
```

Sample `metricbeat.aws.yml` looks like:

```yaml
metricbeat.modules:
- module: aws
  period: 5m
  credential_profile_name: elastic-beats
  shared_credential_file: /usr/share/metricbeat/credentials
  metricsets:
    - ec2
```

* Use AWS credentials in Filebeat configuration

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      access_key_id: '<access_key_id>'
      secret_access_key: '<secret_access_key>'
      session_token: '<session_token>'
    ```

    or

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      access_key_id: '${AWS_ACCESS_KEY_ID:""}'
      secret_access_key: '${AWS_SECRET_ACCESS_KEY:""}'
      session_token: '${AWS_SESSION_TOKEN:""}'
    ```

* Use IAM role ARN

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      role_arn: arn:aws:iam::123456789012:role/test-mb
    ```

* Use shared AWS credentials file

    ```yaml
    filebeat.inputs:
    - type: aws-s3
      queue_url: https://sqs.us-east-1.amazonaws.com/123/test-queue
      credential_profile_name: test-fb
    ```



### AWS Credentials Types [_aws_credentials_types_3]

There are two different types of AWS credentials can be used: access keys and temporary security credentials.

* Access keys

`AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` are the two parts of access keys. They are long-term credentials for an IAM user or the AWS account root user. Please see [AWS Access Keys and Secret Access Keys](https://docs.aws.amazon.com/general/latest/gr/aws-sec-cred-types.html#access-keys-and-secret-access-keys) for more details.

* IAM role ARN

An IAM role is an IAM identity that you can create in your account that has specific permissions that determine what the identity can and cannot do in AWS. A role does not have standard long-term credentials such as a password or access keys associated with it. Instead, when you assume a role, it provides you with temporary security credentials for your role session. IAM role Amazon Resource Name (ARN) can be used to specify which AWS IAM role to assume to generate temporary credentials. Please see [AssumeRole API documentation](https://docs.aws.amazon.com/STS/latest/APIReference/API_AssumeRole.html) for more details.

Here are the steps to set up IAM role using AWS CLI for Metricbeat. Please replace `123456789012` with your own account ID.

Step 1. Create `example-policy.json` file to include all permissions:

```yaml
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Sid": "VisualEditor0",
            "Effect": "Allow",
            "Action": [
                "s3:GetObject",
                "sqs:ReceiveMessage"
            ],
            "Resource": "*"
        },
        {
            "Sid": "VisualEditor1",
            "Effect": "Allow",
            "Action": "sqs:ChangeMessageVisibility",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor2",
            "Effect": "Allow",
            "Action": "sqs:DeleteMessage",
            "Resource": "arn:aws:sqs:us-east-1:123456789012:test-fb-ks"
        },
        {
            "Sid": "VisualEditor3",
            "Effect": "Allow",
            "Action": [
                "sts:AssumeRole",
                "sqs:ListQueues",
                "tag:GetResources",
                "ec2:DescribeInstances",
                "cloudwatch:GetMetricData",
                "ec2:DescribeRegions",
                "iam:ListAccountAliases",
                "sts:GetCallerIdentity",
                "cloudwatch:ListMetrics"
            ],
            "Resource": "*"
        }
    ]
}
```

Step 2. Create IAM policy using the `aws iam create-policy` command:

```bash
$ aws iam create-policy --policy-name example-policy --policy-document file://example-policy.json
```

Step 3. Create the JSON file `example-role-trust-policy.json` that defines the trust relationship of the IAM role

```yaml
{
    "Version": "2012-10-17",
    "Statement": {
        "Effect": "Allow",
        "Principal": { "AWS": "arn:aws:iam::123456789012:root" },
        "Action": "sts:AssumeRole"
    }
}
```

Step 4. Create the IAM role and attach the policy:

```bash
$ aws iam create-role --role-name example-role --assume-role-policy-document file://example-role-trust-policy.json
$ aws iam attach-role-policy --role-name example-role --policy-arn "arn:aws:iam::123456789012:policy/example-policy"
```

After these steps are done, IAM role ARN can be used for authentication in Metricbeat `aws` module.

* Temporary security credentials

Temporary security credentials has a limited lifetime and consists of an access key ID, a secret access key, and a security token which typically returned from `GetSessionToken`. MFA-enabled IAM users would need to submit an MFA code while calling `GetSessionToken`. Please see [Temporary Security Credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_temp.html) for more details. `sts get-session-token` AWS CLI can be used to generate temporary credentials. For example. with MFA-enabled:

```bash
aws> sts get-session-token --serial-number arn:aws:iam::1234:mfa/your-email@example.com --token-code 456789 --duration-seconds 129600
```

Because temporary security credentials are short term, after they expire, the user needs to generate new ones and modify the aws.yml config file with the new credentials. Unless [live reloading](/reference/metricbeat/_live_reloading.md) feature is enabled for Metricbeat, the user needs to manually restart Metricbeat after updating the config file in order to continue collecting Cloudwatch metrics. This will cause data loss if the config file is not updated with new credentials before the old ones expire. For Metricbeat, we recommend users to use access keys in config file to enable aws module making AWS api calls without have to generate new temporary credentials and update the config frequently.

IAM policy is an entity that defines permissions to an object within your AWS environment. Specific permissions needs to be added into the IAM user’s policy to authorize Metricbeat to collect AWS monitoring metrics. Please see documentation under each metricset for required permissions.


## Fields [_fields_7]

For a description of each field in the module, see the [exported fields](/reference/filebeat/exported-fields-awsfargate.md) section.

