FROM alpine:edge

RUN apk add --no-cache stunnel

COPY stunnel.conf /etc/stunnel/stunnel.conf
COPY pki /etc/pki

HEALTHCHECK CMD nc -z localhost 6380
EXPOSE 6380

CMD ["stunnel"]

