resource "local_file" "secrets" {
  content = yamlencode({
    "queue_url" : aws_sqs_queue.filebeat-integtest.url
    "aws_region" : aws_s3_bucket.filebeat-integtest.region
    "bucket_name" : aws_s3_bucket.filebeat-integtest.id
  })
  filename        = "${path.module}/outputs.yml"
  file_permission = "0644"
}
