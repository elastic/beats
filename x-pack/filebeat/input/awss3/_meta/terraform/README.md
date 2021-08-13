# Terraform setup for AWS S3 Input Integration Tests

This directory contains a Terrafrom module that creates the AWS resources needed
for executing the integration tests for the `aws-s3` Filebeat input. It creates
an S3 bucket and SQS queue and configures S3 `ObjectCreated:*` notifications to
be delivered to SQS.

It outputs configuration information that is consumed by the tests to
`outputs.yml`. The AWS resources are randomly named to prevent name collisions
between multiple users.

### Usage

You must have the appropriate AWS environment variables for authentication set
before running Terraform or the integration tests. The AWS key must be
authorized to create and destroy S3 buckets and SQS queues.

1. Execute terraform in this directory to create the resources. This will also
write the `outputs.yml`.

    `terraform apply`


2. View the output configuration and assure the region match in the aws profile used to run
the test or to set the environment variable `AWS_REGION` to the value in the output.

   ```yaml
   "aws_region": "us-east-1"
   "bucket_name": "filebeat-s3-integtest-8iok1h"
   "queue_url": "https://sqs.us-east-1.amazonaws.com/144492464627/filebeat-s3-integtest-8iok1h"
   ```

4. Execute the integration test.

    ```
    cd x-pack/filebeat/inputs/awss3
    go test -tags aws,integration -run TestInputRun.+ -v .
    ```

5. Cleanup AWS resources. Execute terraform to remove the SQS queue and delete
the S3 bucket and its contents.

    `terraform destroy`


