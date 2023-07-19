provider "aws" {
  alias = "localstack"
  access_key                  = "bharat"
  secret_key                  = "bharat"
  region                      = "us-east-1"
  s3_use_path_style           = true
  skip_credentials_validation = true
  skip_metadata_api_check     = true
  skip_requesting_account_id  = true

  endpoints {
    apigateway     = "http://localhost:4566"
    apigatewayv2   = "http://localhost:4566"
    cloudformation = "http://localhost:4566"
    cloudwatch     = "http://localhost:4566"
    dynamodb       = "http://localhost:4566"
    ec2            = "http://localhost:4566"
    es             = "http://localhost:4566"
    elasticache    = "http://localhost:4566"
    firehose       = "http://localhost:4566"
    iam            = "http://localhost:4566"
    kinesis        = "http://localhost:4566"
    lambda         = "http://localhost:4566"
    rds            = "http://localhost:4566"
    redshift       = "http://localhost:4566"
    route53        = "http://localhost:4566"
    s3             = "http://localhost:4566"
    secretsmanager = "http://localhost:4566"
    ses            = "http://localhost:4566"
    sns            = "http://localhost:4566"
    sqs            = "http://localhost:4566"
    ssm            = "http://localhost:4566"
    stepfunctions  = "http://localhost:4566"
    sts            = "http://localhost:4566"
  }
}

resource "random_string" "random_localstack" {
  length  = 6
  special = false
  upper   = false
}

resource "aws_s3_bucket" "filebeat-integtest-localstack" {
  provider = aws.localstack
  bucket        = "filebeat-s3-integtest-localstack-${random_string.random_localstack.result}"
  force_destroy = true
}

resource "aws_sqs_queue" "filebeat-integtest-localstack" {
  provider = aws.localstack
  name   = "filebeat-sqs-integtest-localstack-${random_string.random_localstack.result}"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "sqspolicy",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:*:*:filebeat-sqs-integtest-localstack-${random_string.random_localstack.result}",
      "Condition": {
        "ArnEquals": { "aws:SourceArn": "${aws_s3_bucket.filebeat-integtest-localstack.arn}" }
      }
    }
  ]
}
POLICY

  depends_on = [
    aws_s3_bucket.filebeat-integtest-localstack,
  ]
}

resource "aws_s3_bucket_notification" "bucket_notification-localstack" {
  provider = aws.localstack
  bucket = aws_s3_bucket.filebeat-integtest-localstack.id

  queue {
    queue_arn = aws_sqs_queue.filebeat-integtest-localstack.arn
    events    = ["s3:ObjectCreated:*"]
  }

  depends_on = [
    aws_s3_bucket.filebeat-integtest-localstack,
    aws_sqs_queue.filebeat-integtest-localstack,
  ]
}
