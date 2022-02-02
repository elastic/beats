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

resource "aws_cloudwatch_log_group" "filebeat-integtest-1" {
  name = "filebeat-log-group-integtest-1-${random_string.random.result}"

  tags = {
    Environment = "test"
  }
}

resource "aws_cloudwatch_log_group" "filebeat-integtest-2" {
  name = "filebeat-log-group-integtest-2-${random_string.random.result}"

  tags = {
    Environment = "test"
  }
}

resource "aws_cloudwatch_log_stream" "filebeat-integtest-1" {
  name           = "filebeat-log-stream-integtest-1-${random_string.random.result}"
  log_group_name = aws_cloudwatch_log_group.filebeat-integtest-1.name
}

resource "aws_cloudwatch_log_stream" "filebeat-integtest-2" {
  name           = "filebeat-log-stream-integtest-2-${random_string.random.result}"
  log_group_name = aws_cloudwatch_log_group.filebeat-integtest-2.name
}
