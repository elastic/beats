FROM traefik:1.6
COPY ./traefik.toml /etc/traefik/traefik.toml
HEALTHCHECK --interval=1s --retries=90 CMD curl -f http://localhost:8080/health
