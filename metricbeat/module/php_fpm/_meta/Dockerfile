FROM richarvey/nginx-php-fpm

RUN echo "pm.status_path = /status" >> /etc/php7/php-fpm.d/www.conf
ADD ./php-fpm.conf /etc/nginx/sites-enabled
