build:
	@echo "[-] building fields.yml"
	@cd ../filebeat && mage update
	@echo "[-] filtering fields"
	@python3 filter-fields.py
	@echo "[-] building custom docker image"
	@cd .. && docker build --platform linux/amd64 -t eu.gcr.io/ravelin-builds/sec-tools/filebeat:8.6.1 -f ravelin/Dockerfile .

publish:
	@docker push eu.gcr.io/ravelin-builds/sec-tools/filebeat:8.6.1
