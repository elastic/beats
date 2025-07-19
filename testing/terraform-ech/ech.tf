variable "stack_version" {
  type        = string
  default     = "latest"
  description = "the stack version to use"
}

variable "ech_region" {
  type        = string
  default     = ""
  description = "The region to use"
}

variable "creator" {
  type        = string
  default     = ""
  description = "This is the name who created this deployment"
}

variable "buildkite_id" {
  type        = string
  default     = ""
  description = "The buildkite build id associated with this deployment"
}

variable "pipeline" {
  type        = string
  default     = ""
  description = "The buildkite pipeline slug, useful for in combination with the build id to trace back to the pipeline"
}

variable "deployment_template_id" {
  type        = string
  default     = ""
  description = "The deployment template to use"
}

resource "random_uuid" "deployment_suffix" {
}

# If we have defined a stack version, validate that this version exists on that region and return it.
data "ec_stack" "latest" {
  version_regex = var.stack_version
  region        = local.region
}

locals {
  deployment_name    = join("-", ["beats-ci", substr("${random_uuid.deployment_suffix.result}", 0, 8)])
  deployment_version = data.ec_stack.latest.version

  region                 = coalesce(var.ech_region, "gcp-us-east1")
  deployment_template_id = coalesce(var.deployment_template_id, "gcp-storage-optimized")
}

resource "ec_deployment" "default" {
  name                   = local.deployment_name
  alias                  = local.deployment_name
  region                 = local.region
  deployment_template_id = local.deployment_template_id
  version                = local.deployment_version

  elasticsearch = {
    hot = {
      autoscaling = {}
      size        = "8g"
      zone_count  = 2
    }
  }

  kibana = {
    size       = "1g"
    zone_count = 1
  }
}
