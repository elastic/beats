resource "aws_lb" "test_lb" {
  name            = "${var.elb_name}-lb"
  internal        = false
  security_groups = ["${aws_security_group.allow_http.id}"]
  subnets         = aws_subnet.test_elb.*.id

  depends_on = [
    "aws_internet_gateway.gateway",
    "aws_s3_bucket_policy.write_logs",
  ]

  access_logs {
    enabled = true
    bucket  = "${aws_s3_bucket.test_elb_logs.bucket}"
    prefix  = "httplb"
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = "${aws_lb.test_lb.arn}"
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_lb_target_group.instances.arn}"
  }
}

resource "aws_lb_target_group" "instances" {
  name     = "test-lb-instances"
  port     = 80
  protocol = "HTTP"
  vpc_id   = "${aws_vpc.test_elb.id}"
}

resource "aws_lb_target_group_attachment" "instances" {
  count = length(aws_instance.webserver_backend.*)

  port = 80

  target_id        = "${aws_instance.webserver_backend[count.index].id}"
  target_group_arn = "${aws_lb_target_group.instances.arn}"
}

output "lb_http_address" {
  value = "${aws_lb.test_lb.dns_name}"
}
