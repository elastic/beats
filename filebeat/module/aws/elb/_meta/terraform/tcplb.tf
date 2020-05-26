resource "aws_lb" "test_tcp_lb" {
  name               = "${var.elb_name}-tcp-lb"
  load_balancer_type = "network"
  internal           = false
  subnets            = aws_subnet.test_elb.*.id

  depends_on = [
    "aws_internet_gateway.gateway",
    "aws_s3_bucket_policy.write_logs",
  ]

  access_logs {
    enabled = true
    bucket  = "${aws_s3_bucket.test_elb_logs.bucket}"
    prefix  = "tcplb"
  }
}

resource "aws_lb_listener" "tcp" {
  load_balancer_arn = "${aws_lb.test_tcp_lb.arn}"
  port              = "80"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = "${aws_lb_target_group.tcp_instances.arn}"
  }
}

resource "aws_lb_target_group" "tcp_instances" {
  name     = "test-tcp-lb-instances"
  port     = 80
  protocol = "TCP"
  vpc_id   = "${aws_vpc.test_elb.id}"
}

resource "aws_lb_target_group_attachment" "tcp_instances" {
  count = length(aws_instance.webserver_backend.*)

  port = 80

  target_id        = "${aws_instance.webserver_backend[count.index].id}"
  target_group_arn = "${aws_lb_target_group.tcp_instances.arn}"
}

output "lb_tcp_address" {
  value = "${aws_lb.test_tcp_lb.dns_name}"
}
