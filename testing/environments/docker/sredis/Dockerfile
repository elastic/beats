FROM alpine:edge

RUN apk add --no-cache stunnel

COPY stunnel.conf /etc/stunnel/stunnel.conf
COPY pki /etc/pki

EXPOSE 6380

CMD ["stunnel"]

