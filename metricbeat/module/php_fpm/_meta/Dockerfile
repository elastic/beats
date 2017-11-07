FROM tsouza/nginx-php-fpm:php-7.1

RUN echo "pm.status_path = /status" >> /usr/local/etc/php-fpm.d/www.conf
ADD ./php-fpm.conf /etc/nginx/sites-enabled

HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:81
EXPOSE 81
