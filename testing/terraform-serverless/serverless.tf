variable "region" {
  type        = string
  default     = "aws-us-east-1"
  description = "The serverless region to use"
}

resource "random_uuid" "project_suffix" {
}

locals {
  project_name = join("-", ["beats-ci", substr("${random_uuid.project_suffix.result}", 0, 8)])
}

resource "ec_observability_project" "default" {
  name      = local.project_name
  region_id = var.region
}
