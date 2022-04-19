terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.52"
    }
  }
}

provider "aws" {
  region = var.aws_region
  default_tags {
    tags = {
      Environment = "CI"
      Owner       = "Beats"
      Branch      = var.BRANCH_NAME
      Build       = var.BUILD_ID
      CreatedDate = var.CREATED_DATE
    }
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
    topic_arn     = aws_sns_topic.filebeat-integtest-sns.arn
    events        = ["s3:ObjectCreated:*"]
  }

  depends_on = [
    aws_s3_bucket.filebeat-integtest-sns,
    aws_sns_topic.filebeat-integtest-sns,
  ]
}

resource "aws_sqs_queue" "filebeat-integtest-sns" {
  name   = "filebeat-s3-integtest-sns-${random_string.random.result}"

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
