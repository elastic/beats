FROM centos/systemd:latest AS vm_extension_centos
RUN yum -y install initscripts && yum clean all
RUN yum install sudo wget -y
WORKDIR /sln

COPY ./handler ./handler
COPY ./tests ./tests

