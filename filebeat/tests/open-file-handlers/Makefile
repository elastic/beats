
ES_USER?=admin
ES_PASSWORD?=secure
ES_HOST?=http://localhost:9200/

start:
	# Makes sure images are always built with the most recent snapshot (no caching)
	ES_HOST=${ES_HOST} ES_USER=${ES_USER} ES_PASSWORD=${ES_PASSWORD} docker-compose build --no-cache
	ES_HOST=${ES_HOST} ES_USER=${ES_USER} ES_PASSWORD=${ES_PASSWORD} docker-compose up -d
	ES_HOST=${ES_HOST} ES_USER=${ES_USER} ES_PASSWORD=${ES_PASSWORD} docker-compose scale logs=3

stop:
	docker-compose stop
	docker-compose rm -f
