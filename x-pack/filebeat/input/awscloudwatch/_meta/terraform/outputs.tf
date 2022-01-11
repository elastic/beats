resource "local_file" "secrets" {
  content = yamlencode({
    "log_group_1" : aws_cloudwatch_log_group.filebeat-integtest-1.name
    "log_group_2" : aws_cloudwatch_log_group.filebeat-integtest-2.name
    "log_stream_1" : aws_cloudwatch_log_stream.filebeat-integtest-1.name
    "log_stream_2" : aws_cloudwatch_log_stream.filebeat-integtest-2.name
    "aws_region" : var.aws_region
  })
  filename        = "${path.module}/outputs.yml"
  file_permission = "0644"
}
