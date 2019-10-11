provider "aws" {
  version = "~> 2.8"
  profile = var.profile
  region  = var.region
}

# Needed to access the service arns
data "aws_elb_service_account" "main" {}
