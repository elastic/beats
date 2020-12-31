resource "aws_vpc" "test_eks" {
  cidr_block = "10.0.0.0/16"
}

resource "aws_subnet" "test_eks" {
  count = 2

  vpc_id            = aws_vpc.test_eks.id
  cidr_block        = "10.0.${count.index}.0/24"
  availability_zone = data.aws_availability_zones.available.names[count.index]

  map_public_ip_on_launch = true

  tags = {
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
  }
}

resource "aws_internet_gateway" "gateway" {
  vpc_id = aws_vpc.test_eks.id
}

resource "aws_route_table" "internet_access" {
  vpc_id = aws_vpc.test_eks.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.gateway.id
  }
}

resource "aws_route_table_association" "internet_access" {
  count = length(aws_subnet.test_eks)

  subnet_id      = aws_subnet.test_eks[count.index].id
  route_table_id = aws_route_table.internet_access.id
}
