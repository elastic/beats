# Terraform setup for AWS CloudWatch Input Integration Tests

This directory contains a Terraform module that creates the AWS resources needed
for executing the integration tests for the `aws-cloudwatch` Filebeat input. It
creates two CloudWatch log groups, and one log stream under each log group.

It outputs configuration information that is consumed by the tests to
`outputs.yml`. The AWS resources are randomly named to prevent name collisions
between multiple users.

### Usage

You must have the appropriate AWS environment variables for authentication set
before running Terraform or the integration tests. The AWS key must be
authorized to create and destroy AWS CloudWatch log groups.

1. Initialize a working directory containing Terraform configuration files.

   `terraform init`

2. Execute terraform in this directory to create the resources. This will also
   write the `outputs.yml`. You can use `export TF_VAR_aws_region=NNNNN` in order
   to match the AWS region of the profile you are using.

   `terraform apply`


2. (Optional) View the output configuration.

   ```yaml
    "aws_region": "us-east-1"
    "log_group_name_1": "filebeat-cloudwatch-integtest-1-417koa"
    "log_group_name_2": "filebeat-cloudwatch-integtest-2-417koa"
   ```

3. Execute the integration test.

    ```
    cd x-pack/filebeat/input/awss3
    go test -tags aws,integration -run TestInputRun.+ -v .
    ```

4. Cleanup AWS resources. Execute terraform to delete the log groups created for
testing.

   `terraform destroy`
