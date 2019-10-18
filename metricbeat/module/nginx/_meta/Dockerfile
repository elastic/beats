ARG NGINX_VERSION
FROM nginx:${NGINX_VERSION}
RUN sed -i "/jessie-updates/d" /etc/apt/sources.list
RUN apt-get update && apt-get install -y curl
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost/server-status
COPY ./nginx.conf /etc/nginx/
