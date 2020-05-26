resource "aws_security_group" "allow_http" {
  name        = "allow_http"
  description = "Allow HTTP inbound traffic"
  vpc_id      = "${aws_vpc.test_elb.id}"

  ingress {
    from_port = 80
    to_port   = 81
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0"
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
