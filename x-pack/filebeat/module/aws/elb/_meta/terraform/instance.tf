resource "aws_instance" "webserver_backend" {
  count = length(var.availability_zones)

  ami           = "${data.aws_ami.ubuntu.id}"
  instance_type = "t2.micro"
  subnet_id     = "${aws_subnet.test_elb[count.index].id}"
  user_data     = "${data.local_file.install_webserver.content}"

  associate_public_ip_address = true
  vpc_security_group_ids = [
    "${aws_security_group.allow_http.id}",
  ]
}

data "aws_ami" "ubuntu" {
  most_recent = true

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-bionic-18.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  owners = ["099720109477"] # Canonical
}

provider "local" {
  version = "~> 1.4"
}

data "local_file" "install_webserver" {
  filename = "./install_webserver.sh"
}
