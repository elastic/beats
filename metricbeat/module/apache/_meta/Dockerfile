FROM httpd:2.4.20
RUN apt-get update && apt-get install -y curl
HEALTHCHECK CMD curl -f http://localhost
COPY ./httpd.conf /usr/local/apache2/conf/httpd.conf
