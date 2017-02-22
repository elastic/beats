FROM nginx:1.9
RUN apt-get update && apt-get install -y curl
HEALTHCHECK CMD curl -f http://localhost/server-status
COPY ./nginx.conf /etc/nginx/
