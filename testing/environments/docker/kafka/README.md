# Kafka test container

This Docker container provides an environment for testing with Kafka. It exposes two ports to the host system, `9092` for `PLAINTEXT` and `9093` for `SASL/SSL` with username `beats` and password `KafkaTest`.

## Certificates

The test environment uses a self-signed SSL certificate in the broker. To connect, clients will need to set `certs/client.truststore.jks` as their trust store.

The files in the `certs` directory were generated with these commands:

```sh
# create the broker's key
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias broker -validity 5000 -keyalg RSA -genkey

What is your first and last name?
  [Unknown]:  kafka
  ...

# create a new certificate authority
openssl req -new -x509 -keyout ca-key -out ca-cert -days 5000

# add the CA to the kafka client's trust store
keytool -keystore client.truststore.jks -storepass KafkaTest -alias CARoot -keyalg RSA -import -file ca-cert

# export the server certificate
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias broker -certreq -file broker-cert

# sign it with the CA
openssl x509 -req -CA ca-cert -CAkey ca-key -in broker-cert -out broker-cert-signed -days 5000 -CAcreateserial -passin pass:KafkaTest

# import CA and signed cert back into server keystore
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias CARoot -import -file ca-cert
keytool -keystore broker.keystore.jks -storepass KafkaTest -alias broker -import -file broker-cert-signed

```
