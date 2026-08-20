ARG ELASTICSEARCH_VERSION
FROM docker.elastic.co/elasticsearch/elasticsearch:${ELASTICSEARCH_VERSION}
COPY --chown=elasticsearch:root testing/environments/docker/elasticsearch/roles.yml config/roles.yml
COPY --chown=elasticsearch:root testing/environments/docker/elasticsearch/users config/users
COPY --chown=elasticsearch:root testing/environments/docker/elasticsearch/users_roles config/users_roles
HEALTHCHECK --interval=1s --retries=300 CMD curl -f http://localhost:9200/_license
