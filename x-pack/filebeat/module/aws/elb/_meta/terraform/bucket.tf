resource "aws_s3_bucket" "test_elb_logs" {
  bucket = var.elb_name
  acl    = "private"

  # Bucket can be destroyed with terraform destroy even if it has objects
  force_destroy = true

  policy = "${data.aws_iam_policy_document.s3_bucket_lb_write.json}"
}

resource "aws_sqs_queue" "queue" {
  name = "s3-event-notification-queue"

  policy = "${data.aws_iam_policy_document.sqs_receive_s3_event.json}"
}

data "aws_sqs_queue" "queue" {
  name = "${aws_sqs_queue.queue.name}"
}

output "sqs_queue_url" {
  value = "${data.aws_sqs_queue.queue.url}"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = "${aws_s3_bucket.test_elb_logs.id}"

  queue {
    queue_arn = "${aws_sqs_queue.queue.arn}"
    events    = ["s3:ObjectCreated:*"]
  }
}

data "aws_iam_policy_document" "s3_bucket_lb_write" {
  policy_id = "s3_bucket_logs"

  statement {
    actions   = ["s3:PutObject"]
    resources = ["arn:aws:s3:::${var.elb_name}/*"]

    principals {
      identifiers = ["${data.aws_elb_service_account.main.arn}"]
      type        = "AWS"
    }
  }
}

data "aws_iam_policy_document" "sqs_receive_s3_event" {
  policy_id = "sqs_receive_s3_event"

  statement {
    actions   = ["sqs:SendMessage"]
    resources = ["arn:aws:sqs:*:*:s3-event-notification-queue"]
    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = ["${aws_s3_bucket.test_elb_logs.arn}"]
    }
  }
}
