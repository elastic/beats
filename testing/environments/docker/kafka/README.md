# Kafka test container

This Docker container provides an environment for testing with Kafka. It exposes two ports to the host system, `9092` for `PLAINTEXT` and `9093` for `SASL/SSL` with username `beats` and password `KafkaTest`.

## Certificates

The test environment uses a self-signed SSL certificate in the broker. To connect, clients will need to set `certs/client.truststore.jks` as their trust store.

The files in the `certs` directory were generated with these commands:

```sh
# create the broker's key
keytool -genkeypair -keystore broker.keystore.jks -storepass KafkaTest \
  -alias broker -keyalg RSA -keysize 2048 -validity 5000 \
  -dname "CN=kafka" \
  -ext "SAN=dns:kafka,dns:localhost,ip:127.0.0.1"


What is your first and last name?
  [Unknown]:  kafka
  ...

# create a new certificate authority, use passphrase KafkaTest
openssl req -new -x509 -keyout ca-key -out ca-cert -days 5000

# add the CA to the kafka client's trust store
keytool -keystore client.truststore.jks -storepass KafkaTest -alias CARoot -keyalg RSA -sigalg SHA256withRSA -import -file ca-cert

# export the server certificate
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias broker -certreq -file broker-cert

# sign it with the CA
openssl x509 -req \
  -in broker-cert \
  -CA ca-cert -CAkey ca-key \
  -CAcreateserial \
  -out broker-cert-signed \
  -days 5000 \
  -passin pass:KafkaTest \
  -sha256 \
  -extfile <(printf '%s\n' \
    '[v3_req]' \
    'subjectAltName=DNS:kafka,DNS:localhost,IP:127.0.0.1' \
    'keyUsage=digitalSignature,keyEncipherment' \
    'extendedKeyUsage=serverAuth') \
  -extensions v3_req
  
# import CA and signed cert back into server keystore
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias CARoot -import -file ca-cert
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias broker -import -file broker-cert-signed

```
