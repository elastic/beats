ARG NGINX_VERSION
FROM nginx:${NGINX_VERSION}
HEALTHCHECK --interval=5s --timeout=3s CMD wget -q -O /dev/null http://localhost || exit 1
EXPOSE 80
COPY ./nginx.conf /etc/nginx/
