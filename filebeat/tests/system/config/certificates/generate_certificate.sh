#!/bin/bash
set -x
#

DAYS=36500

rm -rf *.key *.crt *.csr cacert.srl
echo "Generate simple self signed CA"
openssl req -x509  -batch -nodes -newkey rsa:2048 -keyout beats1.key \
  -out beats1.crt -days $DAYS -subj /CN=localhost

openssl req -x509  -batch -nodes -newkey rsa:2048 -keyout beats2.key \
  -out beats2.crt -days $DAYS -subj /CN=localhost

# Generate CA for mutual auth
echo "----"
echo "Generate CACert without a passphrase"
openssl genrsa -out cacert.key 2048
openssl req -sha256 -config cacert.cfg -extensions extensions -new -x509 -days $DAYS -key cacert.key -out cacert.crt \
  -subj "/C=CA/ST=Quebec/L=Montreal/O=beats/OU=root"

echo "----"
echo "Generate client1"
openssl genrsa -out client1.key 2048
openssl req -sha256 -new -key client1.key -out client1.csr \
  -subj "/C=CA/ST=Quebec/L=Montreal/O=beats/OU=server/CN=localhost"
openssl x509 -req -days $DAYS -in client1.csr -CA cacert.crt -CAkey cacert.key -CAcreateserial \
  -out client1.crt\
  -extfile cacert.cfg -extensions server

echo "----"
echo "Generate client2"
openssl genrsa -out client2.key 2048
openssl req -sha256 -new -key client2.key -out client2.csr \
  -subj "/C=CA/ST=Quebec/L=Montreal/O=beats/OU=client/CN=localhost"
openssl x509 -req -days $DAYS -in client2.csr -CA cacert.crt -CAkey cacert.key -CAserial cacert.srl\
  -out client2.crt \
  -extfile cacert.cfg -extensions client

echo "----"
echo "create the certificate chains"
cat cacert.crt >> client1.crt
cat cacert.crt >> client2.crt
