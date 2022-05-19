FROM eclipse-mosquitto:1.6.8
# Silence log spam from the periodic health check below.
RUN sed -i "s|#connection_messages true|connection_messages false|g" /mosquitto/config/mosquitto.conf
HEALTHCHECK --interval=1s --retries=600 CMD nc -z localhost 1883
