ENV?=latest.yml

start:
	# This is run every time to make sure the environment is up-to-date
	docker-compose -f ${ENV} build
	docker-compose -f ${ENV} run beat bash

stop:
	docker-compose -f ${ENV} stop
	docker-compose -f ${ENV} rm -f

# Be careful using this command, as it will remove all containers and volumes of your docker-machine
clean:
	docker stop $(shell docker ps -a -q)
	docker rm -v $(shell docker ps -a -q)
