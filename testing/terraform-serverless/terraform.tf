terraform {
  required_version = ">= 1.2.7"

  required_providers {
    ec = {
      source  = "elastic/ec"
      version = ">= 0.12.2"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.5"
    }
  }
}
