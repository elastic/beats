resource "aws_vpc" "test_elb" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "test_elb" {
  count = length(var.availability_zones)

  vpc_id            = "${aws_vpc.test_elb.id}"
  cidr_block        = "10.0.${count.index}.0/24"
  availability_zone = var.availability_zones[count.index]
}

resource "aws_internet_gateway" "gateway" {
  vpc_id = "${aws_vpc.test_elb.id}"
}

resource "aws_route_table" "internet_access" {
  vpc_id = "${aws_vpc.test_elb.id}"

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = "${aws_internet_gateway.gateway.id}"
  }
}

resource "aws_route_table_association" "internet_access" {
  count = length(var.availability_zones)

  subnet_id      = "${aws_subnet.test_elb[count.index].id}"
  route_table_id = "${aws_route_table.internet_access.id}"
}
