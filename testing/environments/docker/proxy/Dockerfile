FROM alpine:edge

RUN apk add --no-cache squid bash

COPY squid.conf /etc/squid/squid.conf

HEALTHCHECK --interval=1s --retries=600 CMD nc -z localhost 3128
EXPOSE 3128

CMD ["squid", "--foreground"]
