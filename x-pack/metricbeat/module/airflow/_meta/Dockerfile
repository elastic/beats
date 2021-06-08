# Airflow is started to fetch Statsd metrics from it
FROM apache/airflow:2.1.0

ENV DEBIAN_FRONTEND=noninteractive LANGUAGE=C.UTF-8 LANG=C.UTF-8 LC_ALL=C.UTF-8 \
    LC_CTYPE=C.UTF-8 LC_MESSAGES=C.UTF-8

ARG STATSD_HOST
ARG STATSD_PORT

HEALTHCHECK --interval=1s --retries=90 CMD curl -f localhost:8080
EXPOSE 8080

USER root
RUN apt-get update && apt-get install -y --no-install-recommends git

RUN pip install "apache-airflow[statsd]==${AIRFLOW_VERSION}"

USER airflow

ENV AIRFLOW__CORE__LOAD_EXAMPLES=True

RUN airflow db init

RUN airflow users create \
    --username admin \
    --password admin \
    --firstname Admin \
    --lastname Admin \
    --role Admin \
    --email obs-dc-team@elastic.co


RUN sed -i 's/statsd_on = False/statsd_on = True/g' ${AIRFLOW_HOME}/airflow.cfg && \
 sed -i "s/statsd_host = localhost/statsd_host = ${STATSD_HOST}/g" ${AIRFLOW_HOME}/airflow.cfg && \
 sed -i "s/statsd_port = 8125/statsd_port = ${STATSD_PORT}/g" ${AIRFLOW_HOME}/airflow.cfg

ENTRYPOINT ["/bin/bash", "-c", "airflow webserver --port 8080 -D & airflow scheduler"]
