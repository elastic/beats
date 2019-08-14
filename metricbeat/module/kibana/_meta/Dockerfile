FROM docker.elastic.co/kibana/kibana:7.3.0
HEALTHCHECK --interval=1s --retries=300 --start-period=60s CMD python -c 'import urllib, json; response = urllib.urlopen("http://myelastic:changeme@localhost:5601/api/status"); data = json.loads(response.read()); exit(1) if data["status"]["overall"]["state"] != "green" else exit(0);'

