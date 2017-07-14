FROM golang:1.8.3

RUN echo 'deb http://ftp.de.debian.org/debian sid main' >> /etc/apt/sources.list

RUN apt-get update && apt-get install -y auditd && apt-get clean
