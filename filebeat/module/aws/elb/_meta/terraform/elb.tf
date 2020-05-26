resource "aws_elb" "test_elb" {
  name            = "${var.elb_name}-elb"
  internal        = false
  security_groups = ["${aws_security_group.allow_http.id}"]
  subnets         = "${aws_subnet.test_elb.*.id}"

  depends_on = ["aws_internet_gateway.gateway"]

  access_logs {
    enabled       = true
    bucket        = "${aws_s3_bucket.test_elb_logs.bucket}"
    bucket_prefix = "elb"
    interval      = 5 # minutes
  }

  listener {
    instance_port     = 80
    instance_protocol = "http"
    lb_port           = 80
    lb_protocol       = "http"
  }

  listener {
    instance_port     = 80
    instance_protocol = "tcp"
    lb_port           = 81
    lb_protocol       = "tcp"
  }
}

resource "aws_elb_attachment" "instances" {
  count = length(aws_instance.webserver_backend.*)

  instance = "${aws_instance.webserver_backend[count.index].id}"
  elb      = "${aws_elb.test_elb.id}"
}

output "elb_http_address" {
  value = "${aws_elb.test_elb.dns_name}"
}

output "elb_tcp_address" {
  value = "${aws_elb.test_elb.dns_name}:81"
}
