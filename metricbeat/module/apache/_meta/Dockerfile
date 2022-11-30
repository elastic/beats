ARG APACHE_VERSION=${APACHE_VERSION}
FROM httpd:$APACHE_VERSION
RUN apt update && yes | apt install curl
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost
COPY ./httpd.conf /usr/local/apache2/conf/httpd.conf
