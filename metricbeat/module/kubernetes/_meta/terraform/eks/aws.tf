provider "aws" {
  region = "eu-central-1"
}

data "aws_region" "current" {}

data "aws_availability_zones" "available" {
  state = "available"
}

