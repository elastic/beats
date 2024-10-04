FROM alpine:edge

RUN apk add --no-cache squid bash

COPY squid.conf /etc/squid/squid.conf
COPY pki /etc/pki

RUN chmod 600 /etc/squid/squid.conf; \
	chmod 600 /etc/pki/tls/certs/*; \
	chmod 600 /etc/pki/tls/private/*;

HEALTHCHECK --interval=1s --retries=600 CMD nc -z localhost 3128
EXPOSE 3128

CMD ["squid", "--foreground"]
