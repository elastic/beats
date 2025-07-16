FROM gvenzl/oracle-xe:18-slim

# Set environment variables
ENV ORACLE_PASSWORD=Oradoc_db1
ENV ORACLE_DATABASE=XE

# Expose Oracle port
EXPOSE 1521

# Set working directory
WORKDIR /opt/oracle

# Improved healthcheck
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
  CMD echo "SELECT 1 FROM DUAL;" | sqlplus -S system/${ORACLE_PASSWORD}@//localhost:1521/XE | grep -q "^1$" || exit 1
