provider "aws" {
  version = "~> 2.58"
  default_tags {
    tags = {
      environment  = var.ENVIRONMENT
      repo         = var.REPO
      branch       = var.BRANCH
      build        = var.BUILD_ID
      created_date = var.CREATED_DATE
    }
  }
}

provider "random" {
  version = "~> 2.2"
}

resource "random_id" "suffix" {
  byte_length = 4
}

resource "random_password" "db" {
  length  = 16
  special = false
}

resource "aws_db_instance" "test" {
  identifier          = "metricbeat-test-${random_id.suffix.hex}"
  allocated_storage   = 20 // Gigabytes
  engine              = "mysql"
  instance_class      = "db.t2.micro"
  name                = "metricbeattest"
  username            = "foo"
  password            = random_password.db.result
  skip_final_snapshot = true // Required for cleanup
}

resource "aws_sqs_queue" "test" {
  name                      = "metricbeat-test-${random_id.suffix.hex}"
  receive_wait_time_seconds = 10
}

resource "aws_s3_bucket" "test" {
  bucket        = "metricbeat-test-${random_id.suffix.hex}"
  force_destroy = true // Required for cleanup
}

resource "aws_s3_bucket_metric" "test" {
  bucket = aws_s3_bucket.test.id
  name   = "EntireBucket"
}

resource "aws_s3_bucket_object" "test" {
  key     = "someobject"
  bucket  = aws_s3_bucket.test.id
  content = "something"
}

resource "aws_instance" "test" {
  ami           = data.aws_ami.latest-amzn.id
  monitoring    = true
  instance_type = "t2.micro"
  tags = {
    Name = "metricbeat-test"
  }
}

data "aws_ami" "latest-amzn" {
  most_recent = true
  owners      = ["amazon"]
  filter {
    name   = "name"
    values = [
      "amzn2-ami-hvm-*",
    ]
  }
}
