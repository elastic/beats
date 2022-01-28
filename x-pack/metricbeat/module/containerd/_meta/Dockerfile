FROM chanjarster/prometheus-mock-data:latest
COPY containerdv1.5.2.txt /home/java-app/etc/scrape-data.txt
EXPOSE 8080
HEALTHCHECK --interval=1s --retries=90 CMD nc -z localhost 8080
