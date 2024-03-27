ARG NGINX_VERSION
FROM nginx:${NGINX_VERSION}
RUN apt update && apt install -y curl
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost/nginx_status
COPY ./nginx.conf /etc/nginx/
