resource "aws_s3_bucket" "test_elb_logs" {
  bucket = var.bucket_name
  acl    = "private"

  # Bucket can be destroyed with terraform destroy even if it has objects
  force_destroy = true
}

resource "aws_s3_bucket_policy" "write_logs" {
  bucket = "${aws_s3_bucket.test_elb_logs.id}"
  policy = "${data.aws_iam_policy_document.s3_bucket_lb_write.json}"
}

data "aws_iam_policy_document" "s3_bucket_lb_write" {
  policy_id = "s3_bucket_lb_logs"

  # Required by Classic and Application Load Balancers
  statement {
    actions = [
      "s3:PutObject",
    ]
    resources = ["${aws_s3_bucket.test_elb_logs.arn}/*"]

    principals {
      identifiers = ["${data.aws_elb_service_account.main.arn}"]
      type        = "AWS"
    }
  }

  # Network Load Balancers log through delivery.logs.amazonaws.com service
  statement {
    actions = [
      "s3:PutObject",
    ]
    resources = ["${aws_s3_bucket.test_elb_logs.arn}/*"]
    principals {
      identifiers = ["delivery.logs.amazonaws.com"]
      type        = "Service"
    }
  }

  statement {
    actions = [
      "s3:GetBucketAcl"
    ]
    resources = ["${aws_s3_bucket.test_elb_logs.arn}"]
    principals {
      identifiers = ["delivery.logs.amazonaws.com"]
      type        = "Service"
    }
  }
}

output "bucket_name" {
  value = "${aws_s3_bucket.test_elb_logs.bucket}"
}

resource "aws_sqs_queue" "queue" {
  name = var.queue_name
}

resource "aws_sqs_queue_policy" "receive_s3_event" {
  queue_url = "${aws_sqs_queue.queue.id}"
  policy    = "${data.aws_iam_policy_document.sqs_receive_s3_event.json}"
}

data "aws_sqs_queue" "queue" {
  name = "${aws_sqs_queue.queue.name}"
}

output "sqs_queue_url" {
  value = "${data.aws_sqs_queue.queue.url}"
}

resource "aws_s3_bucket_notification" "bucket_notification" {
  bucket = "${aws_s3_bucket.test_elb_logs.id}"

  depends_on = ["aws_sqs_queue_policy.receive_s3_event"]

  queue {
    queue_arn = "${aws_sqs_queue.queue.arn}"
    events    = ["s3:ObjectCreated:*"]
  }
}

data "aws_iam_policy_document" "sqs_receive_s3_event" {
  policy_id = "sqs_receive_s3_event"

  statement {
    actions   = ["sqs:SendMessage"]
    resources = ["${aws_sqs_queue.queue.arn}"]

    principals {
      identifiers = ["*"]
      type        = "AWS"
    }

    condition {
      test     = "ArnEquals"
      variable = "aws:SourceArn"
      values   = ["${aws_s3_bucket.test_elb_logs.arn}"]
    }
  }
}
