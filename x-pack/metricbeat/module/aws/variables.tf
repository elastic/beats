variable "BRANCH_NAME" {
  description = "Branch name for tagging purposes"
  default = "beats"
}

variable "BUILD_ID" {
  description = "Build ID in the CI for tagging purposes"
  default = "unknown-build"
}

variable "CREATED_DATE" {
  description = "Creation date for tagging purposes"
  default = "unknown-build"
}
