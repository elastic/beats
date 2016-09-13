ENV?=latest.yml
BASE_COMMAND=docker-compose -f base.yml -f ports.yml -f ${ENV}

start:
	# This is run every time to make sure the environment is up-to-date
	${BASE_COMMAND} build
	${BASE_COMMAND} run beat bash

stop:
	${BASE_COMMAND} stop
	${BASE_COMMAND} rm -f


up:
	${BASE_COMMAND} build
	${BASE_COMMAND} up

# Be careful using this command, as it will remove all containers and volumes of your docker-machine
clean:
	docker stop $(shell docker ps -a -q)
	docker rm -v $(shell docker ps -a -q)

# Tails the environment log files
logs:
	${BASE_COMMAND} logs -f
