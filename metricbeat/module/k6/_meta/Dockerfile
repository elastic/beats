ARG K6_VERSION
FROM grafana/k6:${K6_VERSION}

COPY ./script.js /home
CMD ["run", "/home/script.js"]

ENV  K6_BROWSER_ENABLED=true
EXPOSE 6565 

HEALTHCHECK --interval=1s --retries=90 CMD wget --spider --server-response http://localhost:6565/v1/metrics || exit 1
