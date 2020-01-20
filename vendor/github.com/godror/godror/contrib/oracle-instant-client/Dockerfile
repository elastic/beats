FROM debian:testing

LABEL maintainer="t.gulacsi@unosoft.hu"

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update && apt-get install -y libaio1 wget unzip

RUN wget -O /tmp/instantclient-basic-linux-x64.zip https://download.oracle.com/otn_software/linux/instantclient/193000/instantclient-basic-linux.x64-19.3.0.0.0dbru.zip

RUN mkdir -p /usr/lib/oracle && unzip /tmp/instantclient-basic-linux-x64.zip -d /usr/lib/oracle

RUN ldconfig -v /usr/lib/oracle/instantclient_19_3
RUN ldd /usr/lib/oracle/instantclient_19_3/libclntsh.so
