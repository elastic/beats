ENV?=snapshot.yml
BASE_COMMAND=docker-compose -f ${ENV} -f local.yml

start:
	# This is run every time to make sure the environment is up-to-date
	${BASE_COMMAND} build --pull --force-rm
	${BASE_COMMAND} run beat bash

stop:
	${BASE_COMMAND} down -v

status:
	${BASE_COMMAND} ps

up:
	${BASE_COMMAND} build
	${BASE_COMMAND} up

up-%:
	${BASE_COMMAND} build $*
	${BASE_COMMAND} up $*

# Be careful using this command, as it will remove all containers and volumes of your docker-machine
clean:
	docker stop $(shell docker ps -a -q)
	docker rm -v $(shell docker ps -a -q)

# Tails the environment log files
logs:
	${BASE_COMMAND} logs -f
