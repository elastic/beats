FROM debian:latest

RUN apt-get update \
  && apt-get upgrade -y \
  && apt-get install stunnel4 -y

COPY stunnel.conf /etc/stunnel/stunnel.conf
COPY pki /etc/pki

EXPOSE 6380

CMD ["stunnel"]

