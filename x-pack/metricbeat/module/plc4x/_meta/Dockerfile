FROM python:3.13-alpine3.22

ARG PYMODBUS_VERSION=3.16.1
RUN pip install --no-cache-dir "pymodbus==${PYMODBUS_VERSION}"

COPY modbus_server.py /modbus_server.py

HEALTHCHECK --interval=2s --retries=30 CMD python -c "import socket; socket.create_connection(('127.0.0.1', 5020), 1)"

EXPOSE 5020
CMD ["python", "/modbus_server.py"]
