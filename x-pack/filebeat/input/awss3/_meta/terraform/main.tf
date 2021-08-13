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
