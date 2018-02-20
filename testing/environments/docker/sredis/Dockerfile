FROM alpine:edge

RUN apk add --no-cache stunnel

COPY stunnel.conf /etc/stunnel/stunnel.conf
COPY pki /etc/pki

RUN chmod 600 /etc/stunnel/stunnel.conf; \
	chmod 600 /etc/pki/tls/certs/*; \
	chmod 600 /etc/pki/tls/private/*;

HEALTHCHECK --interval=1s --retries=600 CMD nc -z localhost 6380
EXPOSE 6380

CMD ["stunnel"]

