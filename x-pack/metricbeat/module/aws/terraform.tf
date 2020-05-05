provider "aws" {
  version = "~> 2.58"
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
