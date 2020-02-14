#!/bin/bash

export CA_SUBJECT='/C=US/ST=California/L=San Francisco/CN=ca@example.com'
export SERVER_SUBJECT='/C=US/ST=California/L=San Francisco/CN=webmaster@example.com'
export CLIENT_SUBJECT='/C=US/ST=California/L=San Francisco/CN=user@example.com'

# certificate authority creation
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 3650 -key ca.key -out ca.crt -subj "$CA_SUBJECT"

# server certificate creation
openssl genrsa -out server.key 1024
openssl req -new -key server.key -out server.csr -subj "$SERVER_SUBJECT"
openssl x509 -req -days 3650 -in server.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out server.crt

# client certificate creation
openssl genrsa -out client.key 1024
openssl req -new -key client.key -out client.csr -subj "$CLIENT_SUBJECT"
openssl x509 -req -days 3650 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 02 -out client.crt

cat server.crt server.key > server.pem
