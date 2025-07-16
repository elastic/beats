FROM gvenzl/oracle-xe:18-slim

# Set environment variables
ENV ORACLE_USERNAME=sys
ENV ORACLE_PASSWORD=Oradoc_db1
ENV ORACLE_DATABASE=XE

# Expose Oracle port
EXPOSE 1521

# Set working directory
WORKDIR /opt/oracle

HEALTHCHECK --interval=30s --timeout=15s --start-period=60s --retries=5 \
  CMD echo "SELECT 1 FROM DUAL;" | sqlplus -S ${ORACLE_USERNAME}/${ORACLE_PASSWORD}@//localhost:1521/XE | grep -q "1" || exit 1
