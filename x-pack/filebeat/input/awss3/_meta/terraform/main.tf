terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.46.0"
    }
  }
}

provider "aws" {
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

resource "random_string" "random" {
  length  = 6
  special = false
  upper   = false
}

resource "aws_s3_bucket" "filebeat-integtest" {
  bucket        = "filebeat-s3-integtest-${random_string.random.result}"
  force_destroy = true
}

resource "aws_sqs_queue" "filebeat-integtest" {
  name   = "filebeat-s3-integtest-${random_string.random.result}"
  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Id": "sqspolicy",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:*:*:filebeat-s3-integtest-${random_string.random.result}",
      "Condition": {
        "ArnEquals": { "aws:SourceArn": "${aws_s3_bucket.filebeat-integtest.arn}" }
      }
    }
  ]
}
POLICY

  depends_on = [
    aws_s3_bucket.filebeat-integtest,
  ]
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = aws_s3_bucket.filebeat-integtest.id

  queue {
    queue_arn = aws_sqs_queue.filebeat-integtest.arn
    events    = ["s3:ObjectCreated:*"]
  }

  depends_on = [
    aws_s3_bucket.filebeat-integtest,
    aws_sqs_queue.filebeat-integtest,
  ]
}

resource "aws_sns_topic" "filebeat-integtest-sns" {
  name = "filebeat-s3-integtest-sns-${random_string.random.result}"

  policy = <<POLICY
{
    "Version":"2012-10-17",
    "Statement":[{
        "Effect": "Allow",
        "Principal": { "Service": "s3.amazonaws.com" },
        "Action": "SNS:Publish",
        "Resource": "arn:aws:sns:*:*:filebeat-s3-integtest-sns-${random_string.random.result}",
        "Condition":{
            "ArnEquals": { "aws:SourceArn": "${aws_s3_bucket.filebeat-integtest-sns.arn}" }
        }
    }]
}
POLICY

  depends_on = [
    aws_s3_bucket.filebeat-integtest-sns,
  ]
}

resource "aws_s3_bucket" "filebeat-integtest-sns" {
  bucket        = "filebeat-s3-integtest-sns-${random_string.random.result}"
  force_destroy = true
}

resource "aws_s3_bucket_notification" "bucket_notification-sns" {
  bucket = aws_s3_bucket.filebeat-integtest-sns.id

  topic {
    topic_arn = aws_sns_topic.filebeat-integtest-sns.arn
    events    = ["s3:ObjectCreated:*"]
  }

  depends_on = [
    aws_s3_bucket.filebeat-integtest-sns,
    aws_sns_topic.filebeat-integtest-sns,
  ]
}

resource "aws_sqs_queue" "filebeat-integtest-sns" {
  name = "filebeat-s3-integtest-sns-${random_string.random.result}"

  policy = <<POLICY
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": "*",
      "Action": "sqs:SendMessage",
      "Resource": "arn:aws:sqs:*:*:filebeat-s3-integtest-sns-${random_string.random.result}",
      "Condition": {
        "ArnEquals": { "aws:SourceArn": "${aws_sns_topic.filebeat-integtest-sns.arn}" }
      }
    }
  ]
}
POLICY

  depends_on = [
    aws_s3_bucket.filebeat-integtest-sns,
    aws_sns_topic.filebeat-integtest-sns
  ]
}

resource "aws_sns_topic_subscription" "filebeat-integtest-sns" {
  topic_arn = aws_sns_topic.filebeat-integtest-sns.arn
  protocol  = "sqs"
  endpoint  = aws_sqs_queue.filebeat-integtest-sns.arn
}
