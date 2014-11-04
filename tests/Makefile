.PHONY: build
build:
	make -C packetbeat

env: env/bin/activate
env/bin/activate: requirements.txt
	test -d env || virtualenv env
	. env/bin/activate; pip install -Ur requirements.txt
	touch env/bin/activate

.PHONY: test
test: build env
	make -C packetbeat test
	. env/bin/activate; nosetests
