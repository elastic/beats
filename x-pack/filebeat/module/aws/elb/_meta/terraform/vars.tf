variable "profile" {
  type    = string
  default = "filebeat"
}

variable "region" {
  type    = string
  default = "eu-central-1"
}

variable "availability_zones" {
  type    = list(string)
  default = ["eu-central-1a", "eu-central-1b"]
}

variable "elb_name" {
  type    = string
  default = "filebeat-aws-elb-test"
}

variable "bucket_name" {
  type    = string
  default = "filebeat-aws-elb-test"
}

variable "queue_name" {
  type    = string
  default = "filebeat-aws-elb-test"
}
