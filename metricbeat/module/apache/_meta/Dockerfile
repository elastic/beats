ARG APACHE_VERSION=2.4.20
FROM httpd:$APACHE_VERSION
RUN sed -i "/jessie-updates/d" /etc/apt/sources.list
RUN apt-get update && apt-get install -y curl
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost
COPY ./httpd.conf /usr/local/apache2/conf/httpd.conf
