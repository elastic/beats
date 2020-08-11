ARG UWSGI_PYTHON_VERSION
FROM python:${UWSGI_PYTHON_VERSION}-alpine

ARG UWSGI_VERSION
RUN apk add --no-cache --virtual .build-deps gcc libc-dev linux-headers curl
RUN pip install --no-cache-dir --trusted-host pypi.python.org uwsgi==${UWSGI_VERSION}

WORKDIR /app
COPY testdata/app /app

HEALTHCHECK --interval=1s --retries=60 --timeout=10s CMD curl http://localhost:8080/
EXPOSE 8080 9191 9192
