ARG NGINX_VERSION
FROM nginx:${NGINX_VERSION}
HEALTHCHECK --interval=5s --timeout=3s CMD curl -f http://localhost/nginx_status || exit 1
EXPOSE 80
COPY ./nginx.conf /etc/nginx/
