FROM docker.elastic.co/beats/filebeat:8.0.0
USER root

# Removing the audit gcp module from the base image
RUN rm -rf /usr/share/filebeat/module/gcp/audit

# Copying new module files into image
COPY x-pack/filebeat/module/gcp/audit /usr/share/filebeat/module/gcp/audit
COPY x-pack/filebeat/module/gcp/cloud_armor /usr/share/filebeat/module/gcp/cloud_armor

# Copying scoped fields
COPY ravelin/fields.yml fields.yml
USER filebeat
